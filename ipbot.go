package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
    "strings"

	"github.com/bwmarrin/discordgo"
)

type commandHandler func(*discordgo.Session, *discordgo.MessageCreate) error

var (
    apiTokenEnv = os.Getenv("IPBOT_TOKEN")
    authUsersEnv = os.Getenv("IPBOT_AUTH_USERS")
    authUsers []string

    commands = map[string]commandHandler {
        "!ping": func(s *discordgo.Session, m *discordgo.MessageCreate) error {
            s.ChannelMessageSend(m.ChannelID, "Pong!")
            return nil
        },
        "!ip": func(s *discordgo.Session, m *discordgo.MessageCreate) error {
            ip, err := getPublicIPv4()
            if err == nil {
                s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Public IP Address: %v", ip))
            }
            
            return err
        },
        "!ipv6": func(s *discordgo.Session, m *discordgo.MessageCreate) error {
            ip, err := getPublicIPv6()
            if err == nil {
                s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Public IP Address: %v", ip))
            }
            
            return err
        },
    }
)

func splitUsers(r rune) bool {
    return r == ':' || r == ',' || r == ';'
}

func main() {
    if len(apiTokenEnv) == 0 {
        log.Fatalln("IPBOT_TOKEN environemt variable not set.")
    }

    authUsers = strings.FieldsFunc(authUsersEnv, splitUsers)
    if len(authUsers) == 0 {
        log.Println("No authorized users registered.")
    }

    dg, err := discordgo.New("Bot " + apiTokenEnv)
    if err != nil {
        log.Fatalln("Error creating Discord session:", err)
    }

    dg.AddHandler(messageCreate)

    dg.Identify.Intents = discordgo.IntentsGuildMessages

    err = dg.Open()
    if err != nil {
        log.Fatalln("Error opening connection:", err)
    }

    log.Println("IpBot is now running. CTRL+C to exit.")
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
    <- sc

    dg.Close()

    log.Println("Shutdown.");
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
    if m.Author.ID == s.State.User.ID {
        return
    }

    if !slices.Contains(authUsers, m.Author.ID) {
        log.Printf("Attempt to call \"%s\" from unauthorized user \"%s\"", m.Content, m.Author.ID)
        return
    }

    log.Printf("Authorized user \"%v\"\n", m.Author)

    handler, has := commands[m.Content]
    if !has {
        return
    }

    log.Printf("Running command \"%s\"\n", m.Content)

    err := handler(s, m)
    if err != nil {
        log.Println("Error running command:", err)
        s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error running command: %v", err))
    }
}

func getPublicIPv4() (string, error) {
    return getPublicIP("api")
}

func getPublicIPv6() (string, error) {
    return getPublicIP("api64")
}

func getPublicIP(api string) (string, error) {
    resp, err := http.Get(fmt.Sprintf("https://%s.ipify.org?format=text", api))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    return string(body), nil
}
