@echo off
echo Запуск телеграм-бота маркетплейса мобильных устройств...
set CGO_ENABLED=1
go run main.go categories.go database.go
pause 