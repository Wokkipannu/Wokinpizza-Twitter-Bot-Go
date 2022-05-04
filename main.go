package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Messages = []Message{}

// Initialize everything
func init() {
	// Load dotenv or log fatal if we fail to load it
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	fmt.Println("Loaded .env file")

	// Connect to the MongoDB
	err2 := mgm.SetDefaultConfig(nil, os.Getenv("DB"), options.Client().ApplyURI(fmt.Sprintf("mongodb://%v", os.Getenv("MONGO_URI"))))
	if err2 != nil {
		log.Fatal(err2)
	}
	fmt.Println("Connected to MongoDB")

	// Load the messages
	messageFile, err3 := os.Open("/twitterbot/message.json")
	if err3 != nil {
		log.Fatal(err3)
	}
	fmt.Println("Successfully Opened message.json")
	defer messageFile.Close()

	jsonParser := json.NewDecoder(messageFile)
	jsonParser.Decode(&Messages)
	fmt.Printf("Successfully Parsed message.json and %v messages\n", len(Messages))
}

// Define the message structure for sending tweets with prefix and suffix
type Message struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

// Define Dailytopping structure for mongodb
type Dailytopping struct {
	mgm.DefaultModel `bson:",inline"`
	Toppings         string `json:"toppings" bson:"toppings"`
}

type ToppingsResponse struct {
	Data    string `json:"data"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Timer for sending toppings to twitter every day at 10AM
const INTERVAL_PERIOD time.Duration = 24 * time.Hour

const HOUR_TO_TICK int = 10
const MINUTE_TO_TICK int = 00
const SECOND_TO_TICK int = 00

type jobTicker struct {
	t *time.Timer
}

func getNextTickDuration() time.Duration {
	now := time.Now()
	nextTick := time.Date(now.Year(), now.Month(), now.Day(), HOUR_TO_TICK, MINUTE_TO_TICK, SECOND_TO_TICK, 0, time.Local)
	if nextTick.Before(now) {
		nextTick = nextTick.Add(INTERVAL_PERIOD)
	}
	return nextTick.Sub(time.Now())
}

func NewJobTicker() jobTicker {
	log.Printf("Started new tick")
	return jobTicker{time.NewTimer(getNextTickDuration())}
}

func (jt jobTicker) updateJobTicker() {
	log.Printf("Started next tick")
	jt.t.Reset(getNextTickDuration())
}

// Main function to use previously created timer
func main() {
	log.Printf("Wokin Pizza Twitter Bot started")
	log.Printf("Sending tweet every day at %v:%v:%v", HOUR_TO_TICK, MINUTE_TO_TICK, SECOND_TO_TICK)

	config := oauth1.NewConfig(os.Getenv("API_KEY"), os.Getenv("API_SECRET"))
	token := oauth1.NewToken(os.Getenv("ACCESS_TOKEN"), os.Getenv("ACCESS_SECRET"))
	httpClient := config.Client(oauth1.NoContext, token)

	client := twitter.NewClient(httpClient)

	rand.Seed(time.Now().UnixNano())

	jt := NewJobTicker()
	for {
		<-jt.t.C

		prefix, toppings, suffix := getTweetData()

		tweet, resp, err := client.Statuses.Update(fmt.Sprintf("%v %v %v", prefix, toppings, suffix), nil)
		if err != nil {
			fmt.Println(err.Error())
		}

		log.Println(fmt.Sprintf("Tweet (%v) created with status code %v!", tweet.ID, resp.StatusCode))

		updateDailyToppings(toppings)

		jt.updateJobTicker()
	}
}

func updateDailyToppings(t string) {
	toppings := &Dailytopping{}
	coll := mgm.Coll(toppings)

	err := coll.First(bson.M{}, toppings)
	if err != nil {
		log.Printf("Failed to fetch daily toppings")
		return
	}

	toppings.Toppings = t
	err2 := mgm.Coll(toppings).Update(toppings)
	if err2 != nil {
		log.Printf("Failed to update daily toppings")
		return
	}

	log.Printf("Updated daily toppings to %v", t)
}

// Get tweet data function
func getTweetData() (string, string, string) {
	res, err := http.Get(os.Getenv("API_URL") + "/random")
	if err != nil {
		log.Print(err)
		return "", "", ""
	}
	defer res.Body.Close()
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Print(readErr)
		return "", "", ""
	}
	toppings := ToppingsResponse{}
	jsonErr := json.Unmarshal(body, &toppings)
	if jsonErr != nil {
		log.Print(jsonErr)
		return "", "", ""
	}

	message := Messages[rand.Intn(len(Messages))]

	return message.Prefix, toppings.Data, message.Suffix
}
