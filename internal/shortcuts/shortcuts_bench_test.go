package shortcuts

import "testing"

var (
	benchDefaultsSink map[string]string
	benchCommandsSink []Command
	benchSharedSink   map[string]string
	benchDefsSink     any
	benchResolveSink  map[string]Chord
)

func BenchmarkDefaults(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		benchDefaultsSink = Defaults()
	}
}

func BenchmarkCommands(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		benchCommandsSink = Commands()
	}
}

func BenchmarkSharedDefaults(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		benchSharedSink = SharedDefaults()
	}
}

func BenchmarkSharedDefinitions(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		benchDefsSink = SharedDefinitions("desktop:windows")
	}
}

func BenchmarkValidateDefaults(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		benchDefaultsSink = Validate(nil)
	}
}

func BenchmarkResolveWarm(b *testing.B) {
	values := map[string]string{}
	_ = Resolve(values)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		benchResolveSink = Resolve(values)
	}
}
