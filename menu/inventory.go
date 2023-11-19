package menu

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/manifoldco/promptui"

	DB "iris/v2/database"
	"iris/v2/models"
	stripeapi "iris/v2/stripe-api"
)

func ListInventory() []string {
	allInventory := []models.Inventory{}
	err := DB.Conn.Find(&allInventory).Error
	if err != nil {
		println("Error: %v\n", err)
		return []string{}
	}
	readableInventory := []string{}
	for _, inv := range allInventory {
		readableInventory = append(readableInventory, fmt.Sprintf("%s (%d left)", inv.DisplayName, inv.Quantity))
	}
	return readableInventory

}

func AddTrackingInventory() {
	invEntry := &models.Inventory{}
	products, err := stripeapi.GetAllProduct(true)
	if err != nil {
		println("There are no active products on Stripe")
		return
	}

	readableFields := []string{}
	for _, product := range products {
		readableFields = append(readableFields, product.Name)
	}
	promptSelect := promptui.Select{
		Label: "Select Product to Track",
		Items: readableFields,
	}
	index, _, err := promptSelect.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	invEntry.DisplayName = products[index].Name
	invEntry.Product = products[index].ID

	prompt := promptui.Prompt{
		Label: "Enter current quantity in stock",
	}
	initialInventory, err := prompt.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	intInventory, err := strconv.Atoi(initialInventory)
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	invEntry.Quantity = int64(intInventory)

	prompt = promptui.Prompt{
		Label: "Track inventory on existing Payment Links that have this item? (y/n)",
	}
	cascadeTrack, _ := prompt.Run()
	if cascadeTrack == "y" {
		allLinks := []models.PaymentLink{}
		err := DB.Conn.Find(&allLinks).Error
		if err != nil {
			println("Error: %v\n", err)
			return
		}
		for _, link := range allLinks {
			prodInLink := stripeapi.GetProductsInLink(link.LinkID)
			if slices.Contains(prodInLink, products[index].ID) {
				// Too lazy to test associations/many2many
				invEntry.PaymentLinkIDs = append(invEntry.PaymentLinkIDs, link.LinkID)
				link.TrackingInventoryIDs = append(link.TrackingInventoryIDs, invEntry.Product)
				DB.Conn.Save(&link)
				println("Now tracking " + products[index].Name + " on link " + link.Nickname)
			}
		}
	}

	DB.Conn.Create(&invEntry)
}

func removeProdFromLinks(inv []string, product string) []string {
	newInv := []string{}
	for _, entry := range inv {
		if entry != product {
			newInv = append(newInv, entry)
		}
	}
	return newInv
}

func RemoveTrackingInventory() {
	allInventory := []models.Inventory{}
	err := DB.Conn.Find(&allInventory).Error
	if err != nil {
		println("Error: %v\n", err)
		return
	}
	readableFields := []string{}
	for _, inv := range allInventory {
		readableFields = append(readableFields, inv.DisplayName)
	}
	promptSelect := promptui.Select{
		Label: "Select Product to Stop Tracking",
		Items: readableFields,
	}
	index, _, err := promptSelect.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}

	for _, linkID := range allInventory[index].PaymentLinkIDs {
		link := models.PaymentLink{}
		DB.Conn.Where(&models.PaymentLink{LinkID: linkID}).First(&link)
		link.TrackingInventoryIDs = removeProdFromLinks(link.TrackingInventoryIDs, allInventory[index].Product)
		DB.Conn.Save(&link)
	}

	DB.Conn.Delete(&allInventory[index])
}

func ChangeInventoryQuantity() {
	allInventory := []models.Inventory{}
	err := DB.Conn.Find(&allInventory).Error
	if err != nil {
		println("Error: %v\n", err)
		return
	}
	readableFields := []string{}
	for _, inv := range allInventory {
		readableFields = append(readableFields, inv.DisplayName)
	}
	promptSelect := promptui.Select{
		Label: "Select Product to Change Quantity",
		Items: readableFields,
	}
	index, _, err := promptSelect.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	prompt := promptui.Prompt{
		Label: "Enter new quantity",
	}
	newQuantity, err := prompt.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	intQuantity, err := strconv.Atoi(newQuantity)
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	allInventory[index].Quantity = int64(intQuantity)
	DB.Conn.Save(&allInventory[index])
}

// Used by webhook server
func UpdateInventory(product string, decrementQuantity int64) {
	inventory := models.Inventory{}
	err := DB.Conn.Where(&models.Inventory{Product: product}).First(&inventory).Error
	if err != nil {
		slog.Error("Error while searching database for product " + product + err.Error())
		return
	}
	inventory.Quantity -= decrementQuantity
	if inventory.Quantity == 0 {
		stripeapi.ToggleProductActivity(product, false)
	}
	DB.Conn.Save(&inventory)
}
