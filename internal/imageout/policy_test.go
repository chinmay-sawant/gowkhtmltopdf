//nolint:testpackage // white-box test exercises the image policy resolver.
package imageout

import (
	"reflect"
	"testing"

	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/settings"
)

func TestImageLoadGlobalUsesOneEffectivePolicy(t *testing.T) {
	t.Parallel()

	global := settings.PdfGlobal{ //nolint:exhaustruct // focused shared load policy
		Load: settings.LoadGlobal{
			Proxy:                 "http://shared-proxy.example",
			Allow:                 []string{"/shared"},
			EnableLocalFileAccess: true,
			NetworkPolicySet:      true,
			NetworkAllowedSchemes: []string{"https"},
			NetworkAllowedHosts:   []string{"shared.example"},
			NetworkBlockPrivate:   true,
			NetworkBlockCrossHost: true,
		},
	}
	image := settings.ImageGlobal{ //nolint:exhaustruct // focused image load policy
		Load: settings.LoadGlobal{ //nolint:exhaustruct // focused image load policy
			Proxy:                 "http://image-proxy.example",
			Allow:                 []string{"/image"},
			NetworkPolicySet:      true,
			NetworkAllowedSchemes: []string{"http"},
			NetworkAllowedHosts:   []string{"image.example"},
		},
	}

	effective := imageLoadGlobal(global, image)
	want := settings.LoadGlobal{
		Proxy:                 "http://image-proxy.example",
		Allow:                 []string{"/image", "/shared"},
		EnableLocalFileAccess: true,
		NetworkPolicySet:      true,
		NetworkAllowedSchemes: []string{"https"},
		NetworkAllowedHosts:   []string{"shared.example"},
		NetworkBlockPrivate:   true,
		NetworkBlockCrossHost: true,
	}

	if !reflect.DeepEqual(effective, want) {
		t.Fatalf("image effective load global = %+v, want %+v", effective, want)
	}

	loader := load.NewLoader(effective)
	if !reflect.DeepEqual(loader.Network.AllowedSchemes, want.NetworkAllowedSchemes) ||
		!reflect.DeepEqual(loader.Network.AllowedHosts, want.NetworkAllowedHosts) ||
		!loader.Network.BlockPrivateNetworks || !loader.Network.BlockCrossHostRedirects {
		t.Fatalf("loader network policy = %+v, want shared restricted policy", loader.Network)
	}
}
