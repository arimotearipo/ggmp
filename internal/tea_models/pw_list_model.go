package teamodels

import (
	"fmt"
	"math"

	"github.com/arimotearipo/ggmp/internal/action"
	"github.com/arimotearipo/ggmp/internal/types"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const MIN_MENU_INDEX = -2

type PasswordsListModel struct {
	searchInput  textinput.Model
	action       *action.Action
	allUris      []types.URI
	filteredUris []types.URI
	selected     int
	operation    string
	result       string
	totalPages   int
	page         int
	limit        int
}

func NewPasswordsListModel(a *action.Action, o string) *PasswordsListModel {
	page, limit := 0, 5
	uris, _ := a.ListURIs()
	totalPages := int(math.Ceil(float64(len(uris)) / float64(limit)))

	searchInput := textinput.New()
	searchInput.Placeholder = "Search"
	searchInput.Focus()

	var filteredUris []types.URI
	if len(uris) > limit {
		filteredUris = uris[0:limit]
	} else {
		filteredUris = uris
	}

	return &PasswordsListModel{
		searchInput:  searchInput,
		action:       a,
		allUris:      uris,
		filteredUris: filteredUris,
		selected:     MIN_MENU_INDEX,
		operation:    o,
		result:       "",
		totalPages:   totalPages,
		page:         page,
		limit:        limit,
	}
}

func (m *PasswordsListModel) handlePagination() {
	startIndex := m.page * m.limit
	endIndex := startIndex + m.limit

	if startIndex >= len(m.allUris) {
		m.allUris = []types.URI{}
		return
	}

	if endIndex > len(m.allUris) {
		endIndex = len(m.allUris)
	}

	m.filteredUris = m.allUris[startIndex:endIndex]
}

func (m *PasswordsListModel) performSearch() {
	m.result = m.searchInput.Value()
	// query := strings.ToLower(m.searchInput.Value())
	// m.filteredUris = []types.URI{}

	// for _, uri := range m.allUris {
	// 	if strings.Contains(strings.ToLower(uri.Uri), query) {
	// 		m.filteredUris = append(m.filteredUris, uri)
	// 	}
	// }

	// m.limit = 3 + len(m.filteredUris)
	// m.selected = 0
}

func (m *PasswordsListModel) handleSelection() tea.Model {
	selectedUri := m.filteredUris[m.selected]

	switch m.operation {
	case "Get password":
		u, p, err := m.action.GetPassword(selectedUri)
		if err != nil {
			m.result = err.Error()
		} else {
			m.result = "Username: " + u + "\nPassword: " + p
		}
	case "Delete password":
		return NewPasswordDeleteModel(m.action, selectedUri)
	case "Update password":
		return NewPasswordUpdateModel(m.action, selectedUri)
	}

	return m
}

func (m *PasswordsListModel) handleMenuNavigation(msg string) {
	switch msg {
	case "up":
		if m.selected-1 < MIN_MENU_INDEX {
			m.selected = len(m.filteredUris) - 1
		} else {
			m.selected--
		}
	case "down":
		if m.selected+1 >= len(m.filteredUris) {
			m.selected = MIN_MENU_INDEX
		} else {
			m.selected++
		}
	}
}

func (m *PasswordsListModel) Init() tea.Cmd {
	m.handlePagination()
	return nil
}

func (m *PasswordsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "down":
			m.handleMenuNavigation(msg.String())

			if m.selected == MIN_MENU_INDEX {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
		case "left", "right":
			if msg.String() == "left" {
				m.page = (m.page - 1 + m.totalPages) % m.totalPages
			} else if msg.String() == "right" {
				m.page = (m.page + 1) % m.totalPages
			}
			m.handlePagination()
		case "enter":
			switch m.selected {
			case -1:
				return NewPasswordMenuModel(m.action), nil
			default:
				return m.handleSelection(), nil
			}
		}
	}

	if m.selected == MIN_MENU_INDEX {
		m.searchInput, _ = m.searchInput.Update(msg)
		m.performSearch()
	}

	return m, nil
}

func (m *PasswordsListModel) View() string {
	s := "Listing saved login details\n"

	for i := MIN_MENU_INDEX; i < len(m.filteredUris); i++ {
		if i == m.selected {
			s += "👉 "
		} else {
			s += "   "
		}

		switch i {
		case MIN_MENU_INDEX:
			s += m.searchInput.View() + "\n"
		case MIN_MENU_INDEX + 1:
			s += "BACK\n"
		default:
			s += m.filteredUris[i].Uri + "\n"
		}
	}
	// just some padding
	for i := 0; i < m.limit-len(m.filteredUris); i++ {
		s += "\n"
	}
	s += fmt.Sprintf("Page %d of %d\n", m.page+1, m.totalPages)

	s += m.result

	return s
}
