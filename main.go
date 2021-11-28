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

// Initialize everything
// Load dotenv or log fatal if we fail to load it
// Connect to the MongoDB
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	err2 := mgm.SetDefaultConfig(nil, os.Getenv("DB"), options.Client().ApplyURI(fmt.Sprintf("mongodb://%v", os.Getenv("MONGO_URI"))))
	if err2 != nil {
		log.Fatal(err2)
	}
}

// Define the message structure for sending tweets with prefix and suffix
type Message struct {
	Prefix string
	Suffix string
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

		// fmt.Println(time.Now(), "- just ticked")

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

	// Defining messages like this for now, later on should probably move on
	// to a database or file so we can change these on the fly without having
	// to change these in the code and restarting the bot
	messages := []Message{
		{
			Prefix: "Kokeile tän päivän komboa 👉",
			Suffix: "🍕",
		},
		{
			Prefix: "Et varmaan uskalla kokeilla 🤏",
			Suffix: "💪",
		},
		{
			Prefix: "Tästä herkulliset täytteet sun pizzaan 🤜",
			Suffix: "🤛",
		},
		{
			Prefix: "Ihan ok, mut ootko kuullut",
			Suffix: "pizzasta? 🙌",
		},
		{
			Prefix: "Nälättääkö? No kokeiles tämmöstä lättyä:",
			Suffix: "🤙",
		},
		{
			Prefix: "Penanaattori suosittelee näitä täytteitä:",
			Suffix: "🍕",
		},
	}

	message := messages[rand.Intn(len(messages))]

	return message.Prefix, toppings.Data, message.Suffix
	// return fmt.Sprintf("%s %s %s", message.Prefix, strings.Join(selectedToppings[:], ", "), message.Suffix)
}
