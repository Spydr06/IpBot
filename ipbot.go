package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
    "slices"
    "strings"

	"github.com/bwmarrin/discordgo"
)

var (
    dmPermission = true
    contexts = []discordgo.InteractionContextType{
        discordgo.InteractionContextPrivateChannel,
        discordgo.InteractionContextGuild,
        discordgo.InteractionContextBotDM,
    }

    integrations = []discordgo.ApplicationIntegrationType{
        discordgo.ApplicationIntegrationGuildInstall,
        discordgo.ApplicationIntegrationUserInstall,
    }

    apiTokenEnv = os.Getenv("IPBOT_TOKEN")
    authUsersEnv = os.Getenv("IPBOT_AUTH_USERS")
    authUsers []string

    commands = []*discordgo.ApplicationCommand{
        {
            Name: "ip",
            Description: "Returns the public IPv4 Address",
            DMPermission: &dmPermission,
            Contexts: &contexts,
            Type: discordgo.ChatApplicationCommand,
            IntegrationTypes: &integrations,
        },
        {
            Name: "ipv6",
            Description: "Returns the public IPv6 Address",
            Contexts: &contexts,
            DMPermission: &dmPermission,
            Type: discordgo.ChatApplicationCommand,
            IntegrationTypes: &integrations,
        },
        {
            Name: "ping",
            Description: "Pings the server",
            Contexts: &contexts,
            DMPermission: &dmPermission,
            Type: discordgo.ChatApplicationCommand,
            IntegrationTypes: &integrations,
        },
    }

    commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        "ip": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
            if err := authorize("ip", i.Interaction); err != nil {
                interactionRespond(s, i.Interaction, "Error: " + err.Error())
                return
            }
            
            ip, err := getPublicIPv4()
            
            var msg string
            if err != nil {
                msg = fmt.Sprintf("Error fetching IPv4: %v", err)
            } else {
                msg = fmt.Sprintf("IPv4: %v", ip)
            }
            
            interactionRespond(s, i.Interaction, msg)
        },
        "ipv6": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
            if err := authorize("ipv6", i.Interaction); err != nil {
                interactionRespond(s, i.Interaction, "Error: " + err.Error())
                return
            }

            ip, err := getPublicIPv6()
            
            var msg string
            if err != nil {
                msg = fmt.Sprintf("Error fetching IPv6: %v", err)
            } else {
                msg = fmt.Sprintf("IPv6: %v", ip)
            }

            interactionRespond(s, i.Interaction, msg)
        },
        "ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
            interactionRespond(s, i.Interaction, "Pong!")
        },
    }
)

func authorize(command string, i *discordgo.Interaction) error {
    user := i.User
    if user == nil {
        user = i.Member.User
    }

    log.Printf("User '%v' (%v) invoked command '%v'", user, user.ID, command)

    if !slices.Contains(authUsers, user.ID) {
        return fmt.Errorf("'@%v' is not in the sudoers file. This incident will be reported.", user)
    }

    log.Printf("Authorized User '%v'", user)

    return nil
}

func interactionRespond(s *discordgo.Session, i *discordgo.Interaction, message string) {
    s.InteractionRespond(i, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: message,
        },
    })
}

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

    dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
        log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
    })

    dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
            h(s, i)
        }
    })

    dg.Identify.Intents = discordgo.IntentsDirectMessages

    err = dg.Open()
    if err != nil {
        log.Fatalln("Error opening connection:", err)
    }
    defer dg.Close()

    log.Printf("Reloading %v (/) commands...", len(commands))

    registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
    for i, v := range commands {
        cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
        if err != nil {
            log.Panicf("Cannot create '%v' command: %v", v.Name, err)
        }
        registeredCommands[i] = cmd
    }

    sc := make(chan os.Signal, 1)
    signal.Notify(sc, os.Interrupt)

    log.Println("IpBot is now running. CTRL+C to exit.")
    
    <- sc

    log.Println("Removing commands...")

/*    for _, v := range registeredCommands {
        err := dg.ApplicationCommandDelete(dg.State.User.ID, "", v.ID)
        if err != nil {
            log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
        }
    }*/

    log.Println("Shutdown.");
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
