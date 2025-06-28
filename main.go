package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"whatsapp-bot/service"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
)

var (
	menuService *service.MenuService
	userStates  = make(map[string]string)
	mu          sync.Mutex
	client      *whatsmeow.Client
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ✅ Load menu dari file JSON
	var err error
	menuService, err = service.NewMenuService("menu/menus.json")
	if err != nil {
		panic(fmt.Errorf("gagal memuat menu: %v", err))
	}

	// ✅ Buat database untuk menyimpan sesi login
	dbLog := waLog.Stdout("Database", "INFO", true)
	container, err := sqlstore.New(ctx, "sqlite3", "file:store.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(fmt.Errorf("gagal membuat store: %v", err))
	}

	// ✅ Ambil device WhatsApp pertama dari database
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(fmt.Errorf("tidak ditemukan device: %v", err))
	}

	// ✅ Buat client WhatsApp
	clientLog := waLog.Stdout("Client", "INFO", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)

	// ✅ Event handler untuk pesan masuk
	client.AddEventHandler(eventHandler)

	// ✅ Login dengan QR Code jika belum pernah login
	if client.Store.ID == nil {
		fmt.Println("QR CODE dibutuhkan. Silakan scan:")
		qrChan, _ := client.GetQRChannel(ctx)
		err = client.Connect()
		if err != nil {
			panic(fmt.Errorf("gagal connect: %v", err))
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
				if evt.Event == "success" {
					break
				}
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(fmt.Errorf("gagal connect: %v", err))
		}
	}

	// ✅ Jalankan terus sampai ditekan CTRL+C
	fmt.Println("Bot berjalan. Tekan CTRL+C untuk keluar.")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nMematikan bot...")
	client.Disconnect()
}

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		handleMessage(v)
	case *events.Connected:
		fmt.Println("Connected to WhatsApp")
	case *events.Disconnected:
		fmt.Println("Disconnected from WhatsApp")
	case *events.LoggedOut:
		fmt.Println("Logged out from WhatsApp")
		os.Exit(1)
	}
}

func handleMessage(v *events.Message) {
	if v.Info.MessageSource.IsFromMe {
		return
	}

	var userMsg string
	if v.Message.GetConversation() != "" {
		userMsg = v.Message.GetConversation()
	} else if v.Message.ExtendedTextMessage != nil {
		userMsg = v.Message.ExtendedTextMessage.GetText()
	} else {
		return
	}

	userJID := v.Info.Sender.ToNonAD().String()
	bareJID := types.NewJID(v.Info.Sender.User, v.Info.Sender.Server)

	mu.Lock()
	state := userStates[userJID]
	resp, nextID := menuService.HandleInput(userMsg, state)
	userStates[userJID] = nextID
	mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ptr := func(s string) *string { return &s }

	_, err := client.SendMessage(ctx, bareJID, &waE2E.Message{
		Conversation: ptr(resp),
	})
	if err != nil {
		fmt.Printf("Gagal kirim pesan ke %s: %v\n", userJID, err)
	}
}
