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
	ID      string   `json:"id"`
	Message string   `json:"message"`
	Options []Option `json:"options"`
}

type MenuService struct {
	Menus map[string]Menu
}

func NewMenuService(path string) (*MenuService, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var menus map[string]Menu
	err = json.Unmarshal(data, &menus)
	if err != nil {
		return nil, err
	}

	return &MenuService{Menus: menus}, nil
}

func (s *MenuService) GetMenu(id string) string {
	menu, ok := s.Menus[id]
	if !ok {
		return "Menu tidak ditemukan. Ketik 'menu' untuk mulai lagi."
	}
	msg := menu.Message
	for _, opt := range menu.Options {
		msg += fmt.Sprintf("\n%s. %s", opt.Value, opt.Label)
	}
	return msg
}

func (s *MenuService) HandleInput(userInput, currentID string) (response, nextID string) {
	if userInput == "menu" {
		return s.GetMenu("0"), "0"
	}

	currentMenu, ok := s.Menus[currentID]
	if !ok {
		return "Menu tidak ditemukan.", "0"
	}

	for _, opt := range currentMenu.Options {
		if opt.Value == userInput {
			return s.GetMenu(opt.Next), opt.Next
		}
	}

	return "Pilihan tidak valid. Silakan coba lagi.", currentID
}
