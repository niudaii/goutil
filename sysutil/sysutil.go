package sysutil

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/zp857/goutil/constants"
	"log"
	"os"
	"runtime"
	"time"
)

func GetCpuPercent() float64 {
	percent, _ := cpu.Percent(time.Second, false)
	return percent[0]
}

func GetMemPercent() float64 {
	memInfo, _ := mem.VirtualMemory()
	return memInfo.UsedPercent
}

func GetDiskPercent() float64 {
	usage, err := disk.Usage("/")
	if err != nil {
		fmt.Println("Failed to get disk usage:", err)
		return 0
	}

	percentage := float64(usage.Used) / float64(usage.Total) * 100
	return percentage
}

var diskIOStats = make([]disk.IOCountersStat, 0)

func loadDiskIO() []disk.IOCountersStat {
	var diskIOList []disk.IOCountersStat
	stats, err := disk.IOCounters()
	if err != nil {
		log.Println(err)
		return diskIOList
	}
	for _, io := range stats {
		diskIOList = append(diskIOList, io)
	}
	return diskIOList
}

func GetDiskIO() (uint64, uint64) {
	if runtime.GOOS == constants.Windows {
		return 0, 0
	}
	// 获取磁盘 I/O 信息
	if len(diskIOStats) == 0 {
		diskIOStats = loadDiskIO()
	}
	time.Sleep(2 * time.Second)
	diskIOStats2 := loadDiskIO()
	// 计算 I/O 占用率
	var readBytes uint64
	var writeBytes uint64
	for _, io2 := range diskIOStats2 {
		for _, io1 := range diskIOStats {
			if io2.Name == io1.Name {
				if io2.ReadBytes != 0 && io1.ReadBytes != 0 && io2.ReadBytes > io1.ReadBytes {
					readBytes += io2.ReadBytes - io1.ReadBytes/2
				}
				if io2.WriteBytes != 0 && io1.WriteBytes != 0 && io2.WriteBytes > io1.WriteBytes {
					writeBytes += io2.WriteBytes - io1.WriteBytes/2
				}
				break
			}
		}
	}
	diskIOStats = diskIOStats2
	return readBytes, writeBytes
}

var netIOStats = make([]net.IOCountersStat, 0)

func loadNetIO() []net.IOCountersStat {
	netStat, _ := net.IOCounters(true)
	netStatAll, _ := net.IOCounters(false)
	var netIOList []net.IOCountersStat
	netIOList = append(netIOList, netStat...)
	netIOList = append(netIOList, netStatAll...)
	return netIOList
}

func GetNetIO() (uint64, uint64) {
	// 获取网络 I/O 信息
	if len(netIOStats) == 0 {
		netIOStats = loadNetIO()
	}
	time.Sleep(2 * time.Second)
	netIOStats2 := loadNetIO()
	// 计算带宽占用
	var bytesSent uint64
	var bytesRecv uint64
	for _, net2 := range netIOStats2 {
		for _, net1 := range netIOStats {
			if net2.Name == net1.Name {
				if net2.BytesSent != 0 && net1.BytesSent != 0 && net2.BytesSent > net1.BytesSent {
					bytesSent += uint64(float64(net2.BytesSent-net1.BytesSent) / 2)
				}
				if net2.BytesRecv != 0 && net1.BytesRecv != 0 && net2.BytesRecv > net1.BytesRecv {
					bytesRecv += uint64(float64(net2.BytesRecv-net1.BytesRecv) / 2)
				}
				break
			}
		}
	}
	return bytesSent, bytesRecv
}

func GetCurrentProcess() (float64, float32) {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		fmt.Printf("Failed to create process handle: %s\n", err)
		return 0, 0
	}
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		fmt.Printf("Failed to get CPU times: %s\n", err)
		return 0, 0
	}
	memPercent, err := proc.MemoryPercent()
	if err != nil {
		fmt.Printf("Failed to get memory info: %s\n", err)
		return 0, 0
	}
	return cpuPercent, memPercent
}
