# CloudMaker

Веб-панель для создания и управления виртуальными машинами через libvirt/QEMU.
Мониторинг ресурсов — Prometheus + Grafana.

## Возможности

- создание VM с cloud-init ISO, CPU, RAM и диском из базового cloud-образа;
- список VM, пауза, продолжение, удаление, сохранение;
- метрики `/metrics` и готовый дашборд Grafana;
- cloud-init, который устанавливает и запускает Chocolate Doom;
- скрипт очистки всех созданных ресурсов.

Домашнее задание: пункты 1, 2, 4, 6, 7, 8, 9, 10, 11, 12.

## Стек

Go, libvirt/QEMU, Docker Compose, Prometheus, Grafana, cloud-init.

## Запуск

```bash
# 1. Базовый cloud-образ (нужен для загрузки VM)
./cloud-init/doom/get-base-image.sh

# 2. ISO с cloud-init для Doom (опционально)
./cloud-init/doom/build.sh

# 3. Весь стек
docker compose up -d --build
```

- Панель: http://localhost:8080
- Grafana: http://localhost:3000 — `admin` / `admin`
- Prometheus: http://localhost:9090

## Doom

В панели создай VM с памятью от 1024 MiB и ISO
`cloud-init/doom/doom-cloud-init.iso`. После запуска узнай VNC-порт:

```bash
virsh vncdisplay <имя-vm>
```

Cloud-init установит Chocolate Doom и shareware WAD, затем запустит игру.
Подключись любым VNC-клиентом к `127.0.0.1:<порт>`.

## Настройки

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `CLOUDMAKER_IMAGE_DIR` | `/home/bobrik/libvirt-images` | ISO и диски VM |
| `CLOUDMAKER_BASE_IMAGE` | `.../ubuntu-base.qcow2` | базовый cloud-образ |
| `CLOUDMAKER_DISK_SIZE` | `10G` | размер диска VM |

Каталог образов должен быть доступен контейнеру и хосту libvirt по одному и тому же пути.

## Структура

- `main.go` — HTTP API и управление libvirt;
- `web/` — веб-интерфейс;
- `monitoring/` — Prometheus и Grafana;
- `cloud-init/doom/` — демо cloud-init;
- `cleanup.sh` — удаление контейнеров, VM, дисков и данных.

## Очистка

```bash
./cleanup.sh
```

Тестовый `cloud-init.iso` в корне можно использовать для проверки cloud-init.
