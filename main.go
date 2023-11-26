package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v76"
	csession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
	"gorm.io/gorm"

	DB "iris/v2/database"
	"iris/v2/menu"
	"iris/v2/models"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		slog.Warn("Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	settings := DB.GetSettings()
	stripe.Key = settings.ApiKey

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), settings.WebhookEndpointSecret)
	if err != nil {
		slog.Error("Error verifying webhook signature: " + err.Error())
		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
		return
	}

	// If error raised while handling event, return bad request.
	if !checkoutWebhookHandler(event) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func checkoutWebhookHandler(event stripe.Event) bool {
	// Handle the checkout.session.completed event
	if event.Type == "checkout.session.completed" {
		var session stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			slog.Error("Error parsing webhook JSON: " + err.Error())
			return false
		}

		if session.PaymentStatus == "paid" {
			activeSessions := &models.ActiveSession{}
			db_res := DB.Conn.First(&activeSessions)
			if db_res.Error != nil {
				if errors.Is(db_res.Error, gorm.ErrRecordNotFound) {
					return true
				}
				slog.Error("Error while reading: " + db_res.Error.Error())
				return false
			}

			internalLinkID := activeSessions.Sessions.Relation[session.ID]
			link := &models.SessionLink{}
			db_res = DB.Conn.Where(&models.SessionLink{LinkID: internalLinkID}).First(&link)
			if db_res.Error != nil {
				if errors.Is(db_res.Error, gorm.ErrRecordNotFound) {
					return true
				}
				slog.Error("Error while reading: " + db_res.Error.Error())
				return false
			}

			link.Used++
			if link.Used == link.MaxUses {
				link.Active = false
			}
			slog.Info("Link " + link.LinkID + " now used " + fmt.Sprintf("%d", link.Used) + " times")
			db_res = DB.Conn.Save(&link)

			if len(link.TrackingInventoryIDs) > 0 {
				// Expand line_items field
				i := csession.ListLineItems(&stripe.CheckoutSessionListLineItemsParams{
					Session: stripe.String(session.ID),
				})
				boughtItems := map[string]int64{}

				for i.Next() {
					boughtItems[i.LineItem().Price.Product.ID] = i.LineItem().Quantity
				}

				for _, trackingItem := range link.TrackingInventoryIDs {
					menu.UpdateInventory(trackingItem, boughtItems[trackingItem])
					slog.Info(fmt.Sprintf("Inventory item %s was bought %d times in link %s", trackingItem, boughtItems[trackingItem], link.LinkID))
				}

				if db_res.Error != nil {
					slog.Error("Error while saving link paid event to database: " + db_res.Error.Error())
					return false
				}

			}

		}
	}

	return true
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<html><body><h1>" + DB.GetSettings().PaymentConfirmationMessage + "</h1></body></html>"))
}

func checkoutSessionHandler(w http.ResponseWriter, r *http.Request) {
	linkID := chi.URLParam(r, "linkID")
	// Get link from database
	session := &models.SessionLink{}
	db_res := DB.Conn.Where(&models.SessionLink{LinkID: linkID}).First(&session)
	if db_res.Error != nil || !session.Active {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Link invalid"))
		return
	}

	stripe.Key = DB.GetSettings().ApiKey
	res, err := csession.New(&session.Params.CheckoutSessionParams)
	if err != nil {
		slog.Error("Error while creating session: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Save session to database
	activeSession := &models.ActiveSession{}
	db_res = DB.Conn.FirstOrCreate(&activeSession)
	if db_res.Error != nil {
		slog.Error("Error while adding to active session: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if activeSession.Sessions.Relation == nil {
		activeSession.Sessions.Relation = map[string]string{}
	}
	activeSession.Sessions.Relation[res.ID] = session.LinkID
	DB.Conn.Save(&activeSession)

	// redirect to session url
	http.Redirect(w, r, res.URL, http.StatusFound)
}

func serveWeb() {
	r := chi.NewRouter()
	r.Get("/pay/{linkID}", checkoutSessionHandler)
	r.Post("/webhook", webhookHandler)
	r.Get("/success", successHandler)
	fmt.Println("Starting server on port 4242")
	http.ListenAndServe(":4242", r)
}

func main() {
	if _, err := os.Stat("./data.db"); err != nil {
		DB.Init()
		menu.ChangeApiKey()
		menu.ChangeEndpointSecret()
		menu.ChangeDomain()
	} else {
		DB.Init()
	}

	if len(os.Args) > 1 && os.Args[1] == "menu" {
		menu.Entry()
	} else {
		serveWeb()
	}

}
