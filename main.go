package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
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

// Define topping structure for mongodb
type Topping struct {
	mgm.DefaultModel `bson:",inline"`
	Topping          string `json:"topping" bson:"topping"`
}

// Function for fetching all toppings from database
func GetAllToppings() ([]Topping, error) {
	result := []Topping{}

	err := mgm.Coll(&Topping{}).SimpleFind(&result, bson.D{})
	if err != nil {
		log.Printf("Error finding toppings")
		return result, err
	}

	return result, nil
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
	fmt.Println("new tick here")
	return jobTicker{time.NewTimer(getNextTickDuration())}
}

func (jt jobTicker) updateJobTicker() {
	fmt.Println("next tick here")
	jt.t.Reset(getNextTickDuration())
}

// Main function to use previously created timer
func main() {
	config := oauth1.NewConfig(os.Getenv("API_KEY"), os.Getenv("API_SECRET"))
	token := oauth1.NewToken(os.Getenv("ACCESS_TOKEN"), os.Getenv("ACCESS_SECRET"))
	httpClient := config.Client(oauth1.NoContext, token)

	client := twitter.NewClient(httpClient)

	rand.Seed(time.Now().UnixNano())
	tweetData := getTweetData()

	tweet, resp, err := client.Statuses.Update(tweetData, nil)
	if err != nil {
		fmt.Println(err.Error())
	}

	log.Println(fmt.Sprintf("Tweet (%v) created with status code %v!", tweet.ID, resp.StatusCode))

	jt := NewJobTicker()
	for {
		<-jt.t.C
		fmt.Println(time.Now(), "- just ticked")
		jt.updateJobTicker()
	}
}

// Get tweet data function
func getTweetData() string {
	toppings, err := GetAllToppings()
	if err != nil {
		fmt.Println(err.Error())
	}

	var selectedToppings []string

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

	for i := 0; i < 4; i++ {
		selectedToppings = append(selectedToppings, toppings[rand.Intn(len(toppings))].Topping)
	}

	message := messages[rand.Intn(len(messages))]

	return fmt.Sprintf("%s %s %s", message.Prefix, strings.Join(selectedToppings[:], ", "), message.Suffix)
}
