# CloudMaker

Веб-панель для создания и управления виртуальными машинами через libvirt/QEMU.
Мониторинг ресурсов — Prometheus + Grafana.

## Возможности

- создание VM с cloud-init ISO, CPU, RAM и диском из базового cloud-образа;
- список VM, пауза, продолжение, удаление, сохранение;
- метрики `/metrics` и готовый дашборд Grafana;
- скрипт очистки всех созданных ресурсов.

Домашнее задание: пункты 1, 2, 4, 6, 7, 8, 9, 10, 11, 12.

## Стек

Go, libvirt/QEMU, Docker Compose, Prometheus, Grafana, cloud-init.

## Запуск

```bash
#1. установить что надо
sudo apt-get install qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virt-manager libvirt-dev

#1.1 утсановить докер(с запуском без root)

# 2. добавить kvm и libvirt в нужную группу
sudo adduser `id -un` libvirt

sudo adduser `id -un` kvm

# 3. Создать каталог для сохранения cloud-init и выдать libvirt нужные права

mkdir -p ~/cloudmaker-images

# 3. Запуск
docker compose up -d --build
```

- Панель: http://localhost:8080
- Grafana: http://localhost:3000 — `admin` / `admin`
- Prometheus: http://localhost:9090

## Структура

- `main.go` — HTTP API и управление libvirt;
- `web/` — веб-интерфейс;
- `monitoring/` — Prometheus и Grafana;
- `cleanup.sh` — удаление контейнеров, VM, дисков и данных.
- `cloud-init.iso` — тестировочный cloud-init

## Очистка

```bash
./cleanup.sh
```

Тестовый `cloud-init.iso` в корне можно использовать для проверки cloud-init.
