@echo off
echo Запуск телеграм-бота маркетплейса мобильных устройств (без БД)...
go run main_sqlite_disabled.go categories.go
pause 