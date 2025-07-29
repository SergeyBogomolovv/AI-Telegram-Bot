package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms/googleai"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	llm, err := googleai.New(
		ctx,
		googleai.WithAPIKey(getEnv("GOOGLE_API_KEY")),
		googleai.WithDefaultModel(getEnv("DEFAULT_MODEL", "gemini-2.5-flash")),
		googleai.WithHarmThreshold(googleai.HarmBlockNone),
	)
	if err != nil {
		log.Fatal(err)
	}

	controller := NewController(getEnv("TELEGRAM_TOKEN"), llm)

	controller.Start()
	log.Println("Bot started. Press Ctrl+C to stop.")
	<-ctx.Done()
	controller.Stop()
}

func init() {
	godotenv.Load()
}

func getEnv(key string, fallback ...string) string {
	value, exists := os.LookupEnv(key)
	if !exists && len(fallback) > 0 {
		return fallback[0]
	}
	return value
}
