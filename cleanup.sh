#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VMS_FILE="$ROOT/vms"
IMAGE_DIR="${VM_IMAGE_DIR:-/home/bobrik/libvirt-images}"
VIRSH_CONN="${VIRSH_CONN:-qemu:///system}"

if [[ "${1:-}" != "--yes" ]]; then
  read -r -p "Удалить контейнеры, VM и все данные CloudMaker? [y/N] " answer
  [[ "$answer" == [yY] ]] || exit 0
fi

virsh_cmd() {
  virsh -c "$VIRSH_CONN" "$@"
}

echo "==> Останавливаю контейнеры, удаляю volumes и образ проекта"
if command -v docker >/dev/null 2>&1 && [[ -f "$ROOT/docker-compose.yml" ]]; then
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$ROOT/docker-compose.yml" down --volumes --remove-orphans --rmi local || true
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose -f "$ROOT/docker-compose.yml" down --volumes --remove-orphans --rmi local || true
  fi
fi

echo "==> Останавливаю и удаляю VM проекта"
if command -v virsh >/dev/null 2>&1; then
  names=()

  if [[ -f "$VMS_FILE" ]]; then
    while IFS= read -r name; do
      [[ -n "$name" ]] && names+=("$name")
    done < "$VMS_FILE"
  fi

  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    if virsh_cmd dumpxml "$name" 2>/dev/null | grep -qF "$IMAGE_DIR/cloud-init-"; then
      names+=("$name")
    fi
  done < <(virsh_cmd list --all --name 2>/dev/null || true)

  if ((${#names[@]})); then
    while IFS= read -r name; do
      echo "  - $name"
      virsh_cmd destroy "$name" >/dev/null 2>&1 || true
      virsh_cmd undefine "$name" --managed-save >/dev/null 2>&1 \
        || virsh_cmd undefine "$name" >/dev/null 2>&1 \
        || true
    done < <(printf '%s\n' "${names[@]}" | sort -u)
  fi
else
  echo "  virsh не найден, пропускаю VM"
fi

echo "==> Удаляю ISO и runtime-файлы проекта"
rm -f "$IMAGE_DIR"/cloud-init-*.iso
rm -rf "$ROOT/tmp" "$ROOT/vms"
touch "$VMS_FILE"
rm -f "$ROOT/test"

echo "Готово: контейнеры, volumes, VM и данные CloudMaker удалены."
