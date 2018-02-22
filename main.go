package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"
)

var (
	addr         = flag.String("addr", "localhost:1004", "http service address")
	redisAddress = flag.String("redis-address", ":6379", "Address to the Redis server")
)

type Message struct {
	Type  string `json:"type"`
	From  string `json:"senderID"`
	To    string `json:"receiverID"`
	Data  string `json:"data"`
	Color string `json:"color"`
}

// type QueryAnswer struct {
// Query  string `json:"query"`
// Answer string `json:"answer"`
// }
//
// type QueryAnswerList struct {
// QueryAnswers []QueryAnswer `json:"list"`
// }

// func GetBotResponses(c redis.Conn) *QueryAnswerList {
// list := &QueryAnswerList{}
// ret, _ := redis.Strings(c.Do("HKEYS", "responses"))
// for _, elem := range ret {
// fmt.Println(elem)
// }
// return list
// }

// responses is a primitive checker that sees what the user sent, and returns a bool (which denotes whether or not the chatbot should respond or not) and the response string
func responses(arg string, c redis.Conn) (bool, string) {
	if arg == "What time is it?" {
		return true, time.Now().Format("3:04PM")
	} else {
		ret, _ := redis.String(c.Do("HGET", "responses", arg))
		if ret != "" {
			return true, ret
		} else {
			return false, ""
		}
	}
}

func respond(c *websocket.Conn, redisconn redis.Conn, msg Message) {
	needresponse, response := responses(msg.Data, redisconn)
	if needresponse {
		// broadcast means it gets sent to everyone
		if msg.Type == "broadcast" {
			c.WriteJSON(&Message{Type: "broadcast", From: "Chatbot", To: "", Data: response, Color: "is-danger"})
			// else is going to handle the case when the message was private
		} else {
			c.WriteJSON(&Message{Type: "private", From: "Chatbot", To: msg.From, Data: response, Color: "is-danger"})
		}
	}

}

// redissetup will test the redis database (specifically the database at 1) for the values that the bot should respond to)
// if any of the values return nil when the program checks with the database, it means they were not set up. In that case, this function will add the entry to the redis database
func redissetup(c redis.Conn) {
	// ret, _ := redis.String(c.Do("HKEYS", "responses"))
	ret, _ := redis.Strings(c.Do("HKEYS", "responses"))

	if len(ret) == 0 {
		_, _ = c.Do("HSET", "responses", "Hi", "Hi, how are you doing?")
		_, _ = c.Do("HSET", "responses", "I am doing fine, How are you?", "I am doing good as well.")
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	redisconn, err := redis.Dial("tcp", ":6379")
	defer redisconn.Close()

	ret, _ := redisconn.Do("SELECT", "1")
	fmt.Printf("%s\n", ret)

	redissetup(redisconn)

	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/mess/ws"}
	log.Printf("connecting to %s", u.String())
	d := websocket.DefaultDialer
	c, _, err := d.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	// conn, _, err2 := websocket.NewClient(c, u, nil, 1024, 1024)
	defer c.Close()
	// fmt.Println("done := make(chan struct{})")

	c.WriteJSON(&Message{Type: "join", From: "Chatbot", To: "", Data: "join", Color: "is-danger"})

	// r := mux.NewRouter()
	// r.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
	// keys := GetBotResponses(redisconn)
	// fmt.Println("keys", keys)
	// })
	//
	// fmt.Println("YOLO")

	// listenandservererr := http.ListenAndServe(":7000", r)
	// if listenandservererr != nil {
	// panic(listenandservererr)
	// }

	// oldmsg := &Message{}
	msg := &Message{}

	c.ReadJSON(msg)
	respond(c, redisconn, *msg)
	// *oldmsg = *msg
	ignoreiteration := false

	for {
		c.ReadJSON(msg)

		if ignoreiteration {
			ignoreiteration = false
			continue
		}
		if msg.Data != "" {
			// if oldmsg.Data != msg.Data && msg.Data != "" {
			// fmt.Println("oldmsg.Data", oldmsg.Data)
			// fmt.Println("msg.Data", msg.Data)
			respond(c, redisconn, *msg)
			// *oldmsg = *msg
			ignoreiteration = true
		}
	}

}

func slugToString(arg string) string {
	return strings.Replace(arg, "_", " ", -1)
}
