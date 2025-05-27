#!/bin/bash

echo "🔧 Оптимизация macOS для максимального RPS"
echo "=========================================="

# Увеличиваем лимиты файловых дескрипторов
echo "📁 Настройка лимитов файловых дескрипторов..."
sudo launchctl limit maxfiles 65536 200000
ulimit -n 65536

# Настройки TCP/IP для высоких нагрузок
echo "🌐 Оптимизация TCP/IP..."
sudo sysctl -w net.inet.tcp.msl=1000
sudo sysctl -w net.inet.tcp.sendspace=65536
sudo sysctl -w net.inet.tcp.recvspace=65536
sudo sysctl -w net.inet.tcp.delayed_ack=0
sudo sysctl -w net.inet.tcp.rfc1323=1
sudo sysctl -w net.inet.tcp.rfc1644=1
sudo sysctl -w net.inet.tcp.always_keepalive=0

# Увеличиваем буферы сокетов
sudo sysctl -w kern.ipc.maxsockbuf=16777216
sudo sysctl -w net.inet.tcp.sockthreshold=64
sudo sysctl -w net.inet.udp.maxdgram=65536

# Оптимизация памяти
echo "💾 Оптимизация памяти..."
sudo sysctl -w vm.pressure_threshold=0.95

# Проверяем текущие лимиты
echo "📊 Текущие лимиты:"
echo "  File descriptors: $(ulimit -n)"
echo "  TCP MSL: $(sysctl -n net.inet.tcp.msl)"
echo "  Send buffer: $(sysctl -n net.inet.tcp.sendspace)"
echo "  Recv buffer: $(sysctl -n net.inet.tcp.recvspace)"

echo "✅ Оптимизация завершена!"
echo "💡 Перезапустите терминал для применения всех изменений" 