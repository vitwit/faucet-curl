package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

type ErrorResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Error   interface{} `json:"error"`
}

type SuccessResponse struct {
	Status bool        `json:"status"`
	Data   interface{} `json:"data"`
}

type BalanceQueryRes struct {
	Balances   []Coins     `json:"balances"`
	Pagination interface{} `json:"pagination"`
}

type Coins struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

var chain, chain2 string
var recaptchaSecretKey string
var amountFaucet, fees1, fees2 string
var amountSteak string
var key string
var node, node2 string
var publicUrl string
var maxTokens float64
var cliName string

const ADDR_LENGTH int = 44

type claim_struct struct {
	Address  string `json:"address"`
	Response string `json:"response"`
}

// Run a background goroutine to remove old entries from the visitors map.
func init() {
	go cleanupVisitors()
}

func getEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		fmt.Println(key, "=", value)
		return value
	} else {
		log.Fatal("Error loading environment variable: ", key)
		return ""
	}
}

func main() {
	err := godotenv.Load(".env.local", ".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	chain = getEnv("FAUCET_CHAIN")
	recaptchaSecretKey = getEnv("FAUCET_RECAPTCHA_SECRET_KEY")
	amountFaucet = getEnv("FAUCET_AMOUNT_FAUCET")
	amountSteak = getEnv("FAUCET_AMOUNT_STEAK")
	key = getEnv("FAUCET_KEY")
	node = getEnv("FAUCET_NODE")
	publicUrl = getEnv("FAUCET_PUBLIC_URL")
	cliName = getEnv("CLI_NAME")
	node2 = getEnv("FAUCET_NODE_2")
	chain2 = getEnv("FAUCET_CHAIN_2")
	fees1 = getEnv("FEES_1")
	fees2 = getEnv("FEES_2")
	maxTokens, err = strconv.ParseFloat(getEnv("MAX_TOKENS_ALLOWED"), 64)
	if err != nil {
		log.Fatal("MAX_TOKENS_ALLOWED value is invalid")
	}

	recaptcha.Init(recaptchaSecretKey)

	r := mux.NewRouter()
	r.HandleFunc("/claim", getCoinsHandler).Methods(http.MethodPost)

	getR := r.Methods(http.MethodGet).Subrouter()
	getR.HandleFunc("/faucet/{address}", curlFaucetHandler).Methods(http.MethodGet)
	getR.Use(limit)

	if err := http.ListenAndServe(publicUrl, r); err != nil {
		log.Fatal("failed to start server", err)
	}
}

func executeCmd(command string, writes ...string) {
	cmd, wc, _ := goExecute(command)

	for _, write := range writes {
		wc.Write([]byte(write + "\n"))
	}
	cmd.Wait()
}

func goExecute(command string) (cmd *exec.Cmd, pipeIn io.WriteCloser, pipeOut io.ReadCloser) {
	cmd = getCmd(command)
	pipeIn, _ = cmd.StdinPipe()
	pipeOut, _ = cmd.StdoutPipe()
	go cmd.Start()
	time.Sleep(time.Second)
	return cmd, pipeIn, pipeOut
}

func getCmd(command string) *exec.Cmd {
	// split command into command and args
	split := strings.Split(command, " ")

	var cmd *exec.Cmd
	if len(split) == 1 {
		cmd = exec.Command(split[0])
	} else {
		cmd = exec.Command(split[0], split[1:]...)
	}

	return cmd
}

func CheckAccountBalance(address, key, nodeAddr, chainId string) error {
	var queryRes BalanceQueryRes
	var balance float64

	command := fmt.Sprintf("%s query bank balances %s --node %v --chain-id %v -o json", cliName, address, nodeAddr, chainId)
	fmt.Println(" command ", command)

	out, accErr := exec.Command("bash", "-c", command).Output()

	if accErr == nil {
		if err := json.Unmarshal(out, &queryRes); err != nil {
			fmt.Printf("Error unmarshalling command line output %v", err)
			return err
		}
	}

	if len(queryRes.Balances) == 0 {
		return nil
	}

	balance, err := strconv.ParseFloat(queryRes.Balances[0].Amount, 64)
	if err != nil {
		return nil
	}

	if balance < maxTokens || accErr != nil {
		return nil
	}

	return errors.New("You have enough tokens in your account")
}

func getCoinsHandler(res http.ResponseWriter, request *http.Request) {
	address := request.FormValue("address")
	captchaResponse := request.FormValue("response")

	fmt.Println("No error", address, captchaResponse)

	(res).Header().Set("Access-Control-Allow-Origin", "*")

	if len(address) != ADDR_LENGTH {
		panic("Invalid address")
	}

	// make sure captcha is valid
	clientIP := realip.FromRequest(request)
	captchaPassed, captchaErr := recaptcha.Confirm(clientIP, captchaResponse)
	if captchaErr != nil {
		panic(captchaErr)
	}

	fmt.Println("Captcha passed? ", captchaPassed)

	if !captchaPassed {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(ErrorResponse{
			Status:  false,
			Message: "Invalid captcha",
		})
		return
	}

	checkAndExecuteTxsHandler(address, res, request)
	return
}

func curlFaucetHandler(res http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	address := vars["address"]
	checkAndExecuteTxsHandler(address, res, request)
	return
}

func checkAndExecuteTxsHandler(address string, res http.ResponseWriter, request *http.Request) {
	var errMsg string
	var isError bool
	//check account balance
	err := CheckAccountBalance(address, key, node, chain)

	if err != nil {
		isError = true
		errMsg = fmt.Sprintf("%s: %s", chain, err.Error())
	} else {
		// send the coins!
		sendFaucet := fmt.Sprintf(
			"%s tx bank send %v %v %v --from %v --node %v --chain-id %v --fees %s --keyring-backend test -y",
			cliName, key, address, amountFaucet, key, node, chain, fees1)
		fmt.Println(time.Now().UTC().Format(time.RFC3339), sendFaucet)

		executeCmd(sendFaucet)
		errMsg = fmt.Sprintf("%s: Successfully sent tokens to  %s", chain, address)
	}

	// Chain 2 faucet
	if node2 != "" {
		//check account balance
		err = CheckAccountBalance(address, key, node2, chain2)

		if err != nil {
			isError = true
			errMsg = fmt.Sprintf("%s, %s: %s", errMsg, chain2, err.Error())
		} else {
			// send the coins!
			sendFaucet := fmt.Sprintf(
				"%s tx bank send %v %v %v --from %v --node %v --chain-id %v --fees %s --keyring-backend test -y",
				cliName, key, address, amountFaucet, key, node2, chain2, fees2)
			fmt.Println(time.Now().UTC().Format(time.RFC3339), sendFaucet)

			executeCmd(sendFaucet)
			errMsg = fmt.Sprintf("%s, %s: Successfully sent tokens to  %s", errMsg, chain2, address)

		}
	}

	// If there is eror in any of chains,then this will be executed
	if isError {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(ErrorResponse{
			Status:  false,
			Message: errMsg,
			Error:   err,
		})
		return
	}

	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(SuccessResponse{
		Status: true,
		Data:   address,
	})

	return
}

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Change the the map to hold values of the type visitor.
var visitors = make(map[string]*visitor)
var mu sync.Mutex

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		// change rate limit here
		// By default, user will have 20 requests and
		// request will be increased for every 15 seconds.
		rt := rate.Every(15 * time.Second)
		limiter := rate.NewLimiter(rt, 20)
		// Include the current time when creating a new visitor.
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		limiter := getVisitor(ip)
		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
