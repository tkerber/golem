// Package xdg wraps around glib's xdg directory functions, to allow retrieval
// of the names of config directories.
package xdg

// #cgo pkg-config: glib-2.0
// #include <glib.h>
import "C"
import "unsafe"

// GetHomeDir gets the current users home directory.
func GetHomeDir() string {
	cstr := C.g_get_home_dir()
	return C.GoString((*C.char)(cstr))
}

// GetUserCacheDir gets the current users cache directory.
func GetUserCacheDir() string {
	cstr := C.g_get_user_cache_dir()
	return C.GoString((*C.char)(cstr))
}

// GetUserConfigDir gets the current users configuration directory.
func GetUserConfigDir() string {
	cstr := C.g_get_user_config_dir()
	return C.GoString((*C.char)(cstr))
}

// GetUserDataDir gets the current users data directory.
func GetUserDataDir() string {
	cstr := C.g_get_user_data_dir()
	return C.GoString((*C.char)(cstr))
}

// GetUserRuntimeDir gets the current users runtime directory.
func GetUserRuntimeDir() string {
	cstr := C.g_get_user_runtime_dir()
	return C.GoString((*C.char)(cstr))
}

// GetUserDesktopDir gets the current users desktop directory.
//
// Or "$HOME/Desktop" if not set.
func GetUserDesktopDir() string {
	cstr := C.g_get_user_special_dir(C.G_USER_DIRECTORY_DESKTOP)
	return C.GoString((*C.char)(cstr))
}

// GetUserDocumentsDir gets the current users documents directory.
//
// May be blank if not set.
func GetUserDocumentsDir() string {
	cstr := C.g_get_user_special_dir(C.G_USER_DIRECTORY_DOCUMENTS)
	return C.GoString((*C.char)(cstr))
}

// GetUserDownloadDir gets the current users downloads directory.
//
// May be blank if not set.
func GetUserDownloadDir() string {
	cstr := C.g_get_user_special_dir(C.G_USER_DIRECTORY_DOWNLOAD)
	return C.GoString((*C.char)(cstr))
}

// GetUserMusicDir gets the current users music directory.
//
// May be blank if not set.
func GetUserMusicDir() string {
	cstr := C.g_get_user_special_dir(C.G_USER_DIRECTORY_MUSIC)
	return C.GoString((*C.char)(cstr))
}

// GetUserPicturesDir gets the current users pictures directory.
//
// May be blank if not set.
func GetUserPicturesDir() string {
	cstr := C.g_get_user_special_dir(C.G_USER_DIRECTORY_PICTURES)
	return C.GoString((*C.char)(cstr))
}

// GetUserPublicShareDir gets the current users public directory.
//
// May be blank if not set.
func GetUserPublicShareDir() string {
	cstr := C.g_get_user_special_dir(C.G_USER_DIRECTORY_PUBLIC_SHARE)
	return C.GoString((*C.char)(cstr))
}

// GetUserTemplatesDir gets the current users templates directory.
//
// May be blank if not set.
func GetUserTemplatesDir() string {
	cstr := C.g_get_user_special_dir(C.G_USER_DIRECTORY_TEMPLATES)
	return C.GoString((*C.char)(cstr))
}

// GetUserVideosDir gets the current users videos directory.
//
// May be blank if not set.
func GetUserVideosDir() string {
	cstr := C.g_get_user_special_dir(C.G_USER_DIRECTORY_VIDEOS)
	return C.GoString((*C.char)(cstr))
}

// GetSystemDataDirs gets a list of the data directories of the system.
func GetSystemDataDirs() []string {
	cdirs := C.g_get_system_data_dirs()
	var dirs []string
	for *cdirs != nil {
		dirs = append(dirs, C.GoString((*C.char)(*cdirs)))
		cdirs = (**C.gchar)(unsafe.Pointer(
			(uintptr(unsafe.Pointer(cdirs))) + unsafe.Sizeof(*cdirs)))
	}
	return dirs
}

// GetSystemConfigDirs gets a list of the config directories of the system.
func GetSystemConfigDirs() []string {
	cdirs := C.g_get_system_config_dirs()
	var dirs []string
	for *cdirs != nil {
		dirs = append(dirs, C.GoString((*C.char)(*cdirs)))
		cdirs = (**C.gchar)(unsafe.Pointer(
			(uintptr(unsafe.Pointer(cdirs))) + unsafe.Sizeof(*cdirs)))
	}
	return dirs
}
