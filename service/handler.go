package service

import (
	"encoding/json"
	"fmt"
	"os"
)

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Next  string `json:"next"`
}

type Menu struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Type     string           `json:"type"` // "options" or "text"
	Message  string           `json:"message"`
	Followup bool             `json:"followup,omitempty"`
	Options  []Option         `json:"options,omitempty"`
	Children map[string]*Menu `json:"children,omitempty"`
}

type MenuService struct {
	Root *Menu
}

func NewMenuService(path string) (*MenuService, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var root Menu
	err = json.Unmarshal(data, &root)
	if err != nil {
		return nil, err
	}

	return &MenuService{Root: &root}, nil
}

// 🔁 Rekursif cari menu berdasarkan ID
func findMenuByID(menu *Menu, id string) *Menu {
	if menu.ID == id {
		return menu
	}
	for _, child := range menu.Children {
		if found := findMenuByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

// 🧠 Handle input user dan cari menu berikutnya
func (s *MenuService) HandleInput(userInput, currentID string) (response, nextID string) {
	// Awal atau reset menu
	if userInput == "menu" || currentID == "" {
		return s.formatMenu(s.Root), s.Root.ID
	}

	current := findMenuByID(s.Root, currentID)
	if current == nil {
		return "Menu tidak ditemukan. Ketik 'menu' untuk memulai ulang.", s.Root.ID
	}

	// Jika tipe text, kembalikan isinya
	if current.Type == "text" {
		return current.Message, current.ID
	}

	// Cocokkan input ke label atau nomor
	for i, opt := range current.Options {
		if userInput == opt.Value || userInput == fmt.Sprintf("%d", i+1) {
			next := findMenuByID(current, opt.Next)
			if next == nil {
				return "Submenu tidak ditemukan.", current.ID
			}
			if next.Type == "text" {
				return next.Message, next.ID
			}
			return s.formatMenu(next), next.ID
		}
	}

	return "Pilihan tidak valid. Silakan coba lagi atau ketik 'menu' untuk kembali ke awal.", current.ID
}

// 🖨️ Format menu sebagai daftar bernomor
func (s *MenuService) formatMenu(menu *Menu) string {
	msg := menu.Message
	for i, opt := range menu.Options {
		msg += fmt.Sprintf("\n%d. %s", i+1, opt.Label)
	}
	return msg
}
