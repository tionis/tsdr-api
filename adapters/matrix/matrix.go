package matrix

import (
	"sync"

	"github.com/tionis/tsdr-api/data" // This implements the glyph specific data layer
)

// Bot represents a config of the Bot
type Bot struct {
	data       *data.GlyphData
	homeServer string
	userName   string
	password   string
}

// Init returns a Bot config object that can then be started
func Init(data *data.GlyphData, homeServer, userName, password string) Bot {
	return Bot{data, homeServer, userName, password}
}

// Start starts the bot adapter with the given data backend
func (b Bot) Start(stop chan bool, syncGroup *sync.WaitGroup) {
	// TODO create channel for receiving messages to send and register it in the dataLayer -> startMessageSendService()
	// TODO pass crypto store db object (sql.db) for crypto store
	// TODO encryption with DB backend, see https://gist.github.com/tionis/be4fb04952dddfcac93398b5747060f6
	// TODO message send channel
	<-stop
	syncGroup.Add(1)
	syncGroup.Done()
	// TODO extract homerserver and userID from matrixUserID
	/*client, err := mautrix.NewClient(b.homeServer, "", "")
	if err != nil {
		panic(err)
	}
	// TODO this is just the copied demo!
	_, err = client.Login(&mautrix.ReqLogin{
		Type:             "m.login.password",
		Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: b.userName},
		Password:         b.password,
		StoreCredentials: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Login successful")

	syncer := client.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, func(source mautrix.EventSource, evt *event.Event) {
		fmt.Printf("<%[1]s> %[4]s (%[2]s/%[3]s)\n", evt.Sender, evt.Type.String(), evt.ID, evt.Content.AsMessage().Body)
	})

	err = client.Sync()
	if err != nil {
		panic(err)
	}*/
}
