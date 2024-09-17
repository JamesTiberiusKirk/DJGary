package bot

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/JamesTiberiusKirk/DJGary/internal/commands/music"
	"github.com/JamesTiberiusKirk/DJGary/internal/config"
	"github.com/JamesTiberiusKirk/DJGary/internal/discord"
	"github.com/JamesTiberiusKirk/DJGary/internal/handlers"
)

func Start() {
	config.Load()
	if config.IsAppEnvironment(config.APP_ENVIRONMENT_TEST) {
		fmt.Println("App environment is test, aborting startup")
		return
	}

	discord.InitSession()
	addHandlers()
	discord.InitConnection()

	music.StartRoutine()

	defer discord.Session.Close()

	fmt.Println("Bot is running. Press Ctrl + C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func addHandlers() {
	// Register handlers as callbacks for the events.
	discord.Session.AddHandler(handlers.ReadyHandler)
	// Session.AddHandler(handlers.GuildCreateHandler)
	discord.Session.AddHandler(handlers.MessageCreateHandler)
}
