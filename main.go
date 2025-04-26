package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	dbPath = "marketplace.db"
)

type Device struct {
	ID          int
	Name        string
	Description string
	Price       float64
	SellerID    int64
	SellerName  string
	Contact     string
	Category    string
}

type User struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
	Contact   string
}

type BotState struct {
	mu           sync.Mutex
	db           *Database
	Devices      []Device
	Users        map[int64]User
	UserStates   map[int64]string
	WaitingInput map[int64]map[string]string
	NextDeviceID int
}

func NewBotState(db *Database) *BotState {
	state := &BotState{
		db:           db,
		Devices:      make([]Device, 0),
		Users:        make(map[int64]User),
		UserStates:   make(map[int64]string),
		WaitingInput: make(map[int64]map[string]string),
		NextDeviceID: 1,
	}

	// Загрузка данных из БД
	state.loadFromDB()

	return state
}

func (bs *BotState) loadFromDB() {
	// Загрузка пользователей
	users, err := bs.db.GetUsers()
	if err != nil {
		log.Printf("Ошибка при загрузке пользователей: %v", err)
	} else {
		bs.Users = users
	}

	// Загрузка устройств
	devices, err := bs.db.GetDevices()
	if err != nil {
		log.Printf("Ошибка при загрузке устройств: %v", err)
	} else {
		bs.Devices = devices
	}

	// Определение следующего ID для устройств
	nextID, err := bs.db.GetNextDeviceID()
	if err != nil {
		log.Printf("Ошибка при получении следующего ID: %v", err)
	} else {
		bs.NextDeviceID = nextID
	}
}

func (bs *BotState) GetDevices() []Device {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.Devices
}

func (bs *BotState) GetDevicesByCategory(category string) []Device {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	devices, err := bs.db.GetDevicesByCategory(category)
	if err != nil {
		log.Printf("Ошибка при получении устройств по категории: %v", err)
		
		// Используем кэш, если не удалось получить из БД
		var categoryDevices []Device
		for _, device := range bs.Devices {
			if device.Category == category {
				categoryDevices = append(categoryDevices, device)
			}
		}
		return categoryDevices
	}
	
	return devices
}

func (bs *BotState) GetUserDevices(userID int64) []Device {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	devices, err := bs.db.GetDevicesByUser(userID)
	if err != nil {
		log.Printf("Ошибка при получении устройств пользователя: %v", err)
		
		// Используем кэш, если не удалось получить из БД
		var userDevices []Device
		for _, device := range bs.Devices {
			if device.SellerID == userID {
				userDevices = append(userDevices, device)
			}
		}
		return userDevices
	}
	
	return devices
}

func (bs *BotState) AddDevice(device Device) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	id, err := bs.db.SaveDevice(device)
	if err != nil {
		log.Printf("Ошибка при сохранении устройства: %v", err)
		
		// Если не удалось сохранить в БД, используем локальное хранение
		device.ID = bs.NextDeviceID
		bs.NextDeviceID++
		bs.Devices = append(bs.Devices, device)
	} else {
		device.ID = id
		bs.Devices = append(bs.Devices, device)
	}
}

func (bs *BotState) RemoveDevice(deviceID int) bool {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	err := bs.db.RemoveDevice(deviceID)
	if err != nil {
		log.Printf("Ошибка при удалении устройства: %v", err)
		
		// Если не удалось удалить из БД, используем локальное хранение
		for i, device := range bs.Devices {
			if device.ID == deviceID {
				bs.Devices = append(bs.Devices[:i], bs.Devices[i+1:]...)
				return true
			}
		}
		return false
	}
	
	// Обновляем локальный кэш
	for i, device := range bs.Devices {
		if device.ID == deviceID {
			bs.Devices = append(bs.Devices[:i], bs.Devices[i+1:]...)
			break
		}
	}
	
	return true
}

func (bs *BotState) FindDeviceByID(deviceID int) (Device, bool) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	device, found, err := bs.db.GetDeviceByID(deviceID)
	if err != nil {
		log.Printf("Ошибка при поиске устройства: %v", err)
		
		// Если не удалось найти в БД, используем локальное хранение
		for _, device := range bs.Devices {
			if device.ID == deviceID {
				return device, true
			}
		}
		return Device{}, false
	}
	
	return device, found
}

func (bs *BotState) SaveUser(user User) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	err := bs.db.SaveUser(user)
	if err != nil {
		log.Printf("Ошибка при сохранении пользователя: %v", err)
	}
	
	bs.Users[user.ID] = user
}

func (bs *BotState) SearchDevices(query string) []Device {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	devices, err := bs.db.SearchDevices(query)
	if err != nil {
		log.Printf("Ошибка при поиске устройств: %v", err)
		
		// Используем кэш, если не удалось получить из БД
		var foundDevices []Device
		lowerQuery := strings.ToLower(query)
		for _, device := range bs.Devices {
			if strings.Contains(strings.ToLower(device.Name), lowerQuery) || 
			   strings.Contains(strings.ToLower(device.Description), lowerQuery) {
				foundDevices = append(foundDevices, device)
			}
		}
		return foundDevices
	}
	
	return devices
}

func (bs *BotState) SetUserState(userID int64, state string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.UserStates[userID] = state
}

func (bs *BotState) GetUserState(userID int64) string {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.UserStates[userID]
}

func (bs *BotState) SetWaitingInput(userID int64, key, value string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if _, ok := bs.WaitingInput[userID]; !ok {
		bs.WaitingInput[userID] = make(map[string]string)
	}
	bs.WaitingInput[userID][key] = value
}

func (bs *BotState) GetWaitingInput(userID int64) map[string]string {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if input, ok := bs.WaitingInput[userID]; ok {
		return input
	}
	return make(map[string]string)
}

func (bs *BotState) ClearWaitingInput(userID int64) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	delete(bs.WaitingInput, userID)
}

func main() {
	// Инициализация базы данных
	db, err := NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Не удалось инициализировать базу данных: %v", err)
	}
	defer db.Close()

	bot, err := tgbotapi.NewBotAPI("ВАШ_ТОКЕН_БОТА")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Бот @%s запущен", bot.Self.UserName)

	state := NewBotState(db)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, update.Message, state)
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update.CallbackQuery, state)
		}
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *BotState) {
	userID := message.From.ID
	userState := state.GetUserState(userID)

	if message.IsCommand() {
		switch message.Command() {
		case "start":
			handleStart(bot, message, state)
		case "help":
			handleHelp(bot, message, state)
		default:
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда. Используйте /help для справки.")
			bot.Send(msg)
		}
		return
	}

	switch userState {
	case "waiting_device_name":
		state.SetWaitingInput(userID, "name", message.Text)
		state.SetUserState(userID, "waiting_device_description")
		
		msg := tgbotapi.NewMessage(message.Chat.ID, "Введите описание устройства:")
		bot.Send(msg)

	case "waiting_device_description":
		state.SetWaitingInput(userID, "description", message.Text)
		state.SetUserState(userID, "waiting_device_price")
		
		msg := tgbotapi.NewMessage(message.Chat.ID, "Введите цену устройства (в рублях):")
		bot.Send(msg)

	case "waiting_device_price":
		state.SetWaitingInput(userID, "price", message.Text)
		state.SetUserState(userID, "waiting_device_contact")
		
		msg := tgbotapi.NewMessage(message.Chat.ID, "Введите контактные данные для связи:")
		bot.Send(msg)

	case "waiting_device_contact":
		state.SetWaitingInput(userID, "contact", message.Text)
		state.SetUserState(userID, "waiting_device_category")
		
		msg := tgbotapi.NewMessage(message.Chat.ID, "Выберите категорию устройства:")
		msg.ReplyMarkup = getCategoryKeyboard()
		bot.Send(msg)

	case "waiting_search_query":
		query := message.Text
		foundDevices := state.SearchDevices(query)
		
		if len(foundDevices) == 0 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Устройства не найдены.")
			msg.ReplyMarkup = getMainKeyboard()
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Найдено устройств: %d", len(foundDevices)))
			msg.ReplyMarkup = getMainKeyboard()
			bot.Send(msg)
			
			for _, device := range foundDevices {
				deviceMsg := tgbotapi.NewMessage(message.Chat.ID, formatDeviceInfo(device))
				bot.Send(deviceMsg)
			}
		}
		
		state.SetUserState(userID, "")

	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Выберите действие:")
		msg.ReplyMarkup = getMainKeyboard()
		bot.Send(msg)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, state *BotState) {
	userID := callbackQuery.From.ID
	data := callbackQuery.Data

	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	bot.Request(callback)

	chatID := callbackQuery.Message.Chat.ID

	if strings.HasPrefix(data, "cat_") {
		categoryCode := strings.TrimPrefix(data, "cat_")
		if userState := state.GetUserState(userID); userState == "waiting_device_category" {
			input := state.GetWaitingInput(userID)
			price := 0.0
			fmt.Sscanf(input["price"], "%f", &price)
			
			device := Device{
				Name:        input["name"],
				Description: input["description"],
				Price:       price,
				SellerID:    userID,
				SellerName:  callbackQuery.From.FirstName,
				Contact:     input["contact"],
				Category:    categoryCode,
			}
			
			state.AddDevice(device)
			state.ClearWaitingInput(userID)
			state.SetUserState(userID, "")
			
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Устройство добавлено!\nНазвание: %s\nОписание: %s\nЦена: %.2f руб.\nКатегория: %s", 
				device.Name, device.Description, device.Price, CategoryNames[device.Category]))
			msg.ReplyMarkup = getMainKeyboard()
			bot.Send(msg)
			return
		} else {
			devices := state.GetDevicesByCategory(categoryCode)
			if len(devices) == 0 {
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("В категории '%s' пока нет устройств.", CategoryNames[categoryCode]))
				msg.ReplyMarkup = getCategoriesKeyboard()
				bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Устройства в категории '%s' (%d):", CategoryNames[categoryCode], len(devices)))
				bot.Send(msg)
				
				for _, device := range devices {
					deviceMsg := tgbotapi.NewMessage(chatID, formatDeviceInfo(device))
					bot.Send(deviceMsg)
				}
				
				backMsg := tgbotapi.NewMessage(chatID, "Выберите другую категорию или вернитесь в главное меню:")
				backMsg.ReplyMarkup = getBackKeyboard()
				bot.Send(backMsg)
			}
			return
		}
	}

	switch data {
	case "browse_devices":
		msg := tgbotapi.NewMessage(chatID, "Выберите категорию:")
		msg.ReplyMarkup = getCategoriesKeyboard()
		bot.Send(msg)

	case "browse_all_devices":
		devices := state.GetDevices()
		if len(devices) == 0 {
			msg := tgbotapi.NewMessage(chatID, "Сейчас нет доступных устройств.")
			msg.ReplyMarkup = getBackKeyboard()
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Доступные устройства (%d):", len(devices)))
			bot.Send(msg)
			
			for _, device := range devices {
				deviceMsg := tgbotapi.NewMessage(chatID, formatDeviceInfo(device))
				bot.Send(deviceMsg)
			}
			
			backMsg := tgbotapi.NewMessage(chatID, "Вернуться к категориям или в главное меню:")
			backMsg.ReplyMarkup = getBackKeyboard()
			bot.Send(backMsg)
		}

	case "sell_device":
		state.SetUserState(userID, "waiting_device_name")
		state.ClearWaitingInput(userID)
		
		msg := tgbotapi.NewMessage(chatID, "Введите название устройства:")
		bot.Send(msg)

	case "my_devices":
		userDevices := state.GetUserDevices(userID)
		if len(userDevices) == 0 {
			msg := tgbotapi.NewMessage(chatID, "У вас пока нет объявлений.")
			msg.ReplyMarkup = getMainKeyboard()
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ваши объявления (%d):", len(userDevices)))
			bot.Send(msg)
			
			for _, device := range userDevices {
				deviceMsg := tgbotapi.NewMessage(chatID, formatDeviceInfo(device))
				deviceMsg.ReplyMarkup = getDeviceActionsKeyboard(device.ID)
				bot.Send(deviceMsg)
			}
			
			backMsg := tgbotapi.NewMessage(chatID, "Вернуться в главное меню:")
			backMsg.ReplyMarkup = getMainMenuButton()
			bot.Send(backMsg)
		}

	case "search_devices":
		state.SetUserState(userID, "waiting_search_query")
		
		msg := tgbotapi.NewMessage(chatID, "Введите поисковый запрос (название или описание):")
		msg.ReplyMarkup = getMainMenuButton()
		bot.Send(msg)

	case "help":
		handleHelp(bot, callbackQuery.Message, state)
		
	case "back_to_main":
		msg := tgbotapi.NewMessage(chatID, "Главное меню:")
		msg.ReplyMarkup = getMainKeyboard()
		bot.Send(msg)
		
	case "back_to_categories":
		msg := tgbotapi.NewMessage(chatID, "Выберите категорию:")
		msg.ReplyMarkup = getCategoriesKeyboard()
		bot.Send(msg)

	default:
		if strings.HasPrefix(data, "remove_device_") {
			idStr := strings.TrimPrefix(data, "remove_device_")
			var deviceID int
			fmt.Sscanf(idStr, "%d", &deviceID)
			
			device, found := state.FindDeviceByID(deviceID)
			if !found {
				msg := tgbotapi.NewMessage(chatID, "Устройство не найдено.")
				msg.ReplyMarkup = getMainKeyboard()
				bot.Send(msg)
				return
			}
			
			if device.SellerID != userID {
				msg := tgbotapi.NewMessage(chatID, "Вы не можете удалить объявление другого пользователя.")
				msg.ReplyMarkup = getMainKeyboard()
				bot.Send(msg)
				return
			}
			
			if state.RemoveDevice(deviceID) {
				msg := tgbotapi.NewMessage(chatID, "Объявление удалено.")
				msg.ReplyMarkup = getMainKeyboard()
				bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(chatID, "Не удалось удалить объявление.")
				msg.ReplyMarkup = getMainKeyboard()
				bot.Send(msg)
			}
		}
	}
}

func handleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *BotState) {
	userID := message.From.ID
	user := User{
		ID:        userID,
		FirstName: message.From.FirstName,
		LastName:  message.From.LastName,
		Username:  message.From.UserName,
	}
	
	state.SaveUser(user)

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Добро пожаловать, %s! Это маркетплейс мобильных устройств. Выберите действие:", message.From.FirstName))
	msg.ReplyMarkup = getMainKeyboard()
	bot.Send(msg)
}

func handleHelp(bot *tgbotapi.BotAPI, message *tgbotapi.Message, state *BotState) {
	helpText := `Доступные действия:

📱 Посмотреть устройства - просмотр всех доступных устройств
💰 Продать устройство - разместить объявление о продаже
🔍 Поиск - поиск устройства по названию или описанию
📋 Мои объявления - просмотр ваших объявлений
ℹ️ Помощь - показать это сообщение

Для начала работы выберите действие на клавиатуре ниже.`

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ReplyMarkup = getMainKeyboard()
	bot.Send(msg)
}

func getCategoryKeyboard() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	
	for _, category := range Categories {
		button := tgbotapi.NewInlineKeyboardButtonData(CategoryNames[category], "cat_"+category)
		row := []tgbotapi.InlineKeyboardButton{button}
		rows = append(rows, row)
	}
	
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func getCategoriesKeyboard() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	
	for _, category := range Categories {
		button := tgbotapi.NewInlineKeyboardButtonData(CategoryNames[category], "cat_"+category)
		row := []tgbotapi.InlineKeyboardButton{button}
		rows = append(rows, row)
	}
	
	allButton := tgbotapi.NewInlineKeyboardButtonData("Все устройства", "browse_all_devices")
	backButton := tgbotapi.NewInlineKeyboardButtonData("« Назад в меню", "back_to_main")
	rows = append(rows, []tgbotapi.InlineKeyboardButton{allButton})
	rows = append(rows, []tgbotapi.InlineKeyboardButton{backButton})
	
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func getBackKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« К категориям", "back_to_categories"),
			tgbotapi.NewInlineKeyboardButtonData("« В главное меню", "back_to_main"),
		),
	)
}

func getMainMenuButton() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« В главное меню", "back_to_main"),
		),
	)
}

func getMainKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Посмотреть устройства", "browse_devices"),
			tgbotapi.NewInlineKeyboardButtonData("💰 Продать устройство", "sell_device"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Поиск", "search_devices"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Мои объявления", "my_devices"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Помощь", "help"),
		),
	)
}

func getDeviceActionsKeyboard(deviceID int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Удалить объявление", fmt.Sprintf("remove_device_%d", deviceID)),
		),
	)
}

func formatDeviceInfo(device Device) string {
	categoryName := CategoryNames[device.Category]
	if categoryName == "" {
		categoryName = "Не указана"
	}
	
	return fmt.Sprintf("📱 *%s*\n📝 %s\n💰 %.2f руб.\n🏷️ %s\n👤 %s\n📞 %s", 
		device.Name, device.Description, device.Price, categoryName, device.SellerName, device.Contact)
} 