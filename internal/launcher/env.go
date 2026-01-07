package launcher

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

// SetupEnvironment configures environment variables for Wayland and GPU compatibility
func (l *Launcher) SetupEnvironment() {
	l.setupWaylandEnv()
	l.setupGPUEnv()
}

// setupWaylandEnv configures environment variables for Wayland compatibility
func (l *Launcher) setupWaylandEnv() {
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")

	if waylandDisplay != "" {
		log.Info("Wayland detected, setting up environment", "display", waylandDisplay)

		// GTK: Prefer Wayland, fall back to X11
		// See: https://wiki.archlinux.org/title/Wayland#GTK
		_ = os.Setenv("GDK_BACKEND", "wayland,x11")

		// Qt: Use Wayland with X11 fallback
		// See: https://wiki.archlinux.org/title/Wayland#Qt
		_ = os.Setenv("QT_QPA_PLATFORM", "wayland;xcb")

		// WebKitGTK: Disable compositing for better Wayland compatibility
		_ = os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")

		log.Debug("Wayland environment variables set",
			"GDK_BACKEND", "wayland,x11",
			"QT_QPA_PLATFORM", "wayland;xcb",
			"WEBKIT_DISABLE_COMPOSITING_MODE", "1",
		)
	} else {
		log.Debug("Not running on Wayland")
	}
}

// setupGPUEnv detects GPU vendor and sets appropriate environment variables
func (l *Launcher) setupGPUEnv() {
	gpuVendor := detectGPUVendor()

	switch gpuVendor {
	case "amd":
		log.Info("AMD GPU detected, applying optimizations")

		// Use RADV (Mesa Vulkan driver) for AMD GPUs
		// See: https://wiki.archlinux.org/title/Vulkan#Switching
		_ = os.Setenv("AMD_VULKAN_ICD", "RADV")

		// Enable GPL (Graphics Pipeline Library) for faster shader compilation
		// See: https://wiki.archlinux.org/title/AMDGPU#ACO_compiler
		_ = os.Setenv("RADV_PERFTEST", "gpl")

		log.Debug("AMD GPU environment set",
			"AMD_VULKAN_ICD", "RADV",
			"RADV_PERFTEST", "gpl",
		)

	case "nvidia":
		log.Info("NVIDIA GPU detected, applying optimizations")

		// Force GBM backend for NVIDIA (required for Wayland on NVIDIA >= 495)
		// See: https://wiki.archlinux.org/title/Wayland#Requirements
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			_ = os.Setenv("GBM_BACKEND", "nvidia-drm")
			_ = os.Setenv("__GLX_VENDOR_LIBRARY_NAME", "nvidia")

			log.Debug("NVIDIA Wayland environment set",
				"GBM_BACKEND", "nvidia-drm",
				"__GLX_VENDOR_LIBRARY_NAME", "nvidia",
			)
		}

	case "intel":
		log.Info("Intel GPU detected")
		// Intel generally works well with defaults

	default:
		log.Debug("Unknown GPU vendor, using defaults")
		// Apply safe defaults that work for most GPUs
		_ = os.Setenv("AMD_VULKAN_ICD", "RADV")
	}
}

// detectGPUVendor attempts to detect the GPU vendor from /sys
func detectGPUVendor() string {
	// Check common GPU vendor IDs in sysfs
	vendorPaths := []string{
		"/sys/class/drm/card0/device/vendor",
		"/sys/class/drm/card1/device/vendor",
	}

	for _, path := range vendorPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		vendor := strings.TrimSpace(string(data))
		switch vendor {
		case "0x1002": // AMD
			return "amd"
		case "0x10de": // NVIDIA
			return "nvidia"
		case "0x8086": // Intel
			return "intel"
		}
	}

	// Fallback: check for loaded kernel modules
	modules, err := os.ReadFile("/proc/modules")
	if err == nil {
		moduleStr := string(modules)
		if strings.Contains(moduleStr, "amdgpu") || strings.Contains(moduleStr, "radeon") {
			return "amd"
		}
		if strings.Contains(moduleStr, "nvidia") {
			return "nvidia"
		}
		if strings.Contains(moduleStr, "i915") {
			return "intel"
		}
	}

	return "unknown"
}
