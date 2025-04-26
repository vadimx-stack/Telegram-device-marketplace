package main

const (
	CategorySmartphone = "smartphone"
	CategoryTablet     = "tablet"
	CategorySmartwatch = "smartwatch"
	CategoryAccessory  = "accessory"
	CategoryOther      = "other"
)

var CategoryNames = map[string]string{
	CategorySmartphone: "Смартфоны",
	CategoryTablet:     "Планшеты",
	CategorySmartwatch: "Умные часы",
	CategoryAccessory:  "Аксессуары",
	CategoryOther:      "Другое",
}

var Categories = []string{
	CategorySmartphone,
	CategoryTablet,
	CategorySmartwatch,
	CategoryAccessory,
	CategoryOther,
} 