package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gilgameshskytrooper/sendmail"
	"github.com/gorilla/websocket"
)

var (
	addr         = flag.String("addr", "localhost:1012", "http service address")
	redisAddress = flag.String("redis-address", ":6379", "Address to the Redis server")
)

type Message struct {
	DeliveryID string `json:"toid"`
	SenderID   string `json:"senderid"`
	Content    string `json:"content"`
}

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
	needresponse, response := responses(msg.Content, redisconn)
	if needresponse {
		// broadcast means it gets sent to everyone
		// c.WriteJSON(&Message{Type: "private", From: "Chatbot", To: msg.From, Data: response, Color: "is-warning"})
		c.WriteJSON(&Message{DeliveryID: msg.SenderID, SenderID: "Chatbot", Content: response})
	} else {
		sendmail.SendEmail("leeas@stolaf.edu", "Andrew's chatbot got an unidentified message", "Andrew's chatbot got an unidentified message.\n\nThe message that was received was \n\n\""+msg.Content+"\".\n\nThis was sent from the user "+msg.SenderID+".\n\nPlease add the appropriate response at https://chatbot.gilgameshskytrooper.io.")
		sendmail.SendEmail("noor1@stolaf.edu", "Andrew's chatbot got an unidentified message", "Andrew's chatbot got an unidentified message.\n\nThe message that was received was \n\n\""+msg.Content+"\".\n\nThis was sent from the user "+msg.SenderID+".\n\nPlease add the appropriate response at https://chatbot.gilgameshskytrooper.io.")
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

func checkIfResponse(arg string, c redis.Conn) bool {
	responses, _ := redis.Strings(c.Do("HKEYS", "responses"))
	for _, elem := range responses {
		answer, _ := redis.String(c.Do("HGET", "responses", elem))
		if answer == arg {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	redisconn, err := redis.Dial("tcp", ":6379")
	defer redisconn.Close()

	ret, _ := redisconn.Do("SELECT", "1")
	fmt.Printf("%s\n", ret)

	redissetup(redisconn)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws/Chatbot"}
	log.Printf("connecting to %s", u.String())
	d := websocket.DefaultDialer
	c, _, err := d.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	msg := &Message{}

	c.ReadJSON(msg)
	respond(c, redisconn, *msg)

	for {
		c.ReadJSON(msg)
		if msg.SenderID == "Chatbot" {
			fmt.Println(msg)
			continue
		}
		if msg.Content != "" {
			respond(c, redisconn, *msg)
		}
	}

}

func slugToString(arg string) string {
	return strings.Replace(arg, "_", " ", -1)
}
