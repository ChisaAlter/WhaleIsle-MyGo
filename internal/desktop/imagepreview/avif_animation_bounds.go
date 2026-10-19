package imagepreview

import (
	"encoding/binary"
	"errors"
	"math"
	"time"
)

type avifSequence struct {
	width, height, count int
	loops, orientation   int
	delays               []time.Duration
	alpha                bool
}

type avifSequenceTrack struct {
	avifSequence
	id, auxiliaryFor, premultipliedBy uint32
	duration                          uint64
}

// IsAnimatedAVIF follows libavif/Chromium AUTO source selection: an explicit
// avif major brand prefers the primary still item; avis prefers the tracks.
// This is detection only; every box/sample is validated before decode.
func IsAnimatedAVIF(data []byte) bool {
	if len(data) < 16 || string(data[4:8]) != "ftyp" {
		return false
	}
	n := uint64(binary.BigEndian.Uint32(data[:4]))
	if n < 16 || n > uint64(len(data)) || (n-16)%4 != 0 || string(data[8:12]) == "avif" {
		return false
	}
	if string(data[8:12]) == "avis" {
		return true
	}
	for p := 16; p < int(n); p += 4 {
		if string(data[p:p+4]) == "avis" {
			return true
		}
	}
	return false
}

func validateAVIFSequence(data []byte) (*avifSequence, error) {
	if err := rasterInput(data); err != nil {
		return nil, err
	}
	if !IsAnimatedAVIF(data) {
		return nil, errors.New("AVIF sequence brand is missing")
	}
	budget := 256
	boxes, err := avifBoxes(data, 0, &budget)
	if err != nil {
		return nil, err
	}
	var movie avifBox
	var media []avifBox
	metadata := 0
	seen := map[string]bool{}
	for _, box := range boxes {
		switch box.kind {
		case "ftyp", "moov", "meta":
			if seen[box.kind] {
				return nil, errors.New("duplicate AVIF sequence metadata")
			}
			seen[box.kind] = true
			metadata += len(box.data)
			if metadata > 1<<20 {
				return nil, errors.New("AVIF sequence metadata exceeds the byte budget")
			}
			if box.kind == "moov" {
				movie = box
			}
			if box.kind == "meta" {
				if err := avifSequenceMeta(box, &budget); err != nil {
					return nil, err
				}
			}
		case "mdat":
			if len(box.data) == 0 {
				return nil, errors.New("empty AVIF sequence media")
			}
			media = append(media, box)
		case "free", "skip":
		default:
			return nil, errors.New("unsupported AVIF sequence top-level box")
		}
	}
	if !seen["moov"] || len(media) == 0 {
		return nil, errors.New("incomplete AVIF sequence")
	}
	children, err := avifBoxes(movie.data, movie.start, &budget)
	if err != nil {
		return nil, err
	}
	var tracks []*avifSequenceTrack
	for _, box := range children {
		if box.kind == "trak" {
			if len(tracks) >= 2 {
				return nil, errors.New("AVIF sequence has extra image tracks")
			}
			track, err := avifParseSequenceTrack(box, media, &budget)
			if err != nil {
				return nil, err
			}
			tracks = append(tracks, track)
		}
	}
	var colorTrack *avifSequenceTrack
	for _, track := range tracks {
		if track.auxiliaryFor == 0 {
			if colorTrack != nil {
				return nil, errors.New("ambiguous AVIF color tracks")
			}
			colorTrack = track
		}
	}
	if colorTrack == nil {
		return nil, errors.New("AVIF sequence has no color track")
	}
	for _, track := range tracks {
		if track == colorTrack {
			continue
		}
		if track.id == colorTrack.id || track.auxiliaryFor != colorTrack.id || track.width != colorTrack.width || track.height != colorTrack.height || track.count != colorTrack.count {
			return nil, errors.New("AVIF alpha track differs from color sequence")
		}
		colorTrack.alpha = true
	}
	if colorTrack.premultipliedBy != 0 {
		if len(tracks) != 2 {
			return nil, errors.New("AVIF sequence references a missing alpha track")
		}
		for _, track := range tracks {
			if track != colorTrack && track.id != colorTrack.premultipliedBy {
				return nil, errors.New("invalid AVIF premultiplied track")
			}
		}
	}
	return &colorTrack.avifSequence, nil
}

// The still metadata is not used as a decoded frame. Its bounded box tree is
// still checked before libavif parses it inside the same isolated WASM heap.
func avifSequenceMeta(box avifBox, budget *int) error {
	if len(box.data) < 4 || binary.BigEndian.Uint32(box.data[:4]) != 0 {
		return errors.New("invalid AVIF sequence item metadata")
	}
	children, err := avifBoxes(box.data[4:], box.start+4, budget)
	if err != nil {
		return err
	}
	for _, child := range children {
		switch child.kind {
		case "iprp", "ipco":
			if err := avifSequencePropertyTree(child, budget); err != nil {
				return err
			}
		case "iinf":
			if len(child.data) < 6 {
				return errors.New("truncated AVIF item information")
			}
			skip := 6
			count := uint32(binary.BigEndian.Uint16(child.data[4:6]))
			if child.data[0] == 1 {
				if len(child.data) < 8 {
					return errors.New("truncated AVIF item information")
				}
				skip = 8
				count = binary.BigEndian.Uint32(child.data[4:8])
			} else if child.data[0] != 0 {
				return errors.New("unsupported AVIF item information")
			}
			if count > MaxFrames {
				return errors.New("AVIF item count exceeds the frame limit")
			}
			items, err := avifBoxes(child.data[skip:], child.start+uint64(skip), budget)
			if err != nil {
				return err
			}
			if uint32(len(items)) != count {
				return errors.New("invalid AVIF item count")
			}
		}
	}
	return nil
}

func avifSequencePropertyTree(box avifBox, budget *int) error {
	children, err := avifBoxes(box.data, box.start, budget)
	if err != nil {
		return err
	}
	for _, child := range children {
		if child.kind == "ipco" {
			if err := avifSequencePropertyTree(child, budget); err != nil {
				return err
			}
		}
	}
	return nil
}

func avifSequenceFields(box avifBox, budget *int) (map[string]avifBox, error) {
	children, err := avifBoxes(box.data, box.start, budget)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]avifBox, len(children))
	for _, child := range children {
		if _, ok := fields[child.kind]; ok {
			return nil, errors.New("duplicate AVIF sequence box")
		}
		fields[child.kind] = child
	}
	return fields, nil
}

func avifParseSequenceTrack(box avifBox, media []avifBox, budget *int) (*avifSequenceTrack, error) {
	f, err := avifSequenceFields(box, budget)
	if err != nil {
		return nil, err
	}
	t := &avifSequenceTrack{}
	h := f["tkhd"].data
	if len(h) < 4 {
		return nil, errors.New("missing AVIF track header")
	}
	switch h[0] {
	case 0:
		if len(h) != 84 {
			return nil, errors.New("invalid AVIF track header")
		}
		t.id = binary.BigEndian.Uint32(h[12:16])
		t.duration = uint64(binary.BigEndian.Uint32(h[20:24]))
		if t.duration == math.MaxUint32 {
			t.duration = math.MaxUint64
		}
	case 1:
		if len(h) != 96 {
			return nil, errors.New("invalid AVIF track header")
		}
		t.id = binary.BigEndian.Uint32(h[20:24])
		t.duration = binary.BigEndian.Uint64(h[28:36])
	default:
		return nil, errors.New("unsupported AVIF track header version")
	}
	if t.id == 0 {
		return nil, errors.New("AVIF track has no identity")
	}
	t.width = int(binary.BigEndian.Uint32(h[len(h)-8:]) >> 16)
	t.height = int(binary.BigEndian.Uint32(h[len(h)-4:]) >> 16)
	pixels, err := rasterPixels(uint64(t.width), uint64(t.height))
	if err != nil {
		return nil, err
	}
	t.loops, err = avifSequenceLoops(f["edts"], t.duration, budget)
	if err != nil {
		return nil, err
	}
	if references, ok := f["tref"]; ok {
		refs, err := avifSequenceFields(references, budget)
		if err != nil {
			return nil, err
		}
		for kind, ref := range refs {
			if len(ref.data) != 4 {
				return nil, errors.New("invalid AVIF track reference")
			}
			switch kind {
			case "auxl":
				t.auxiliaryFor = binary.BigEndian.Uint32(ref.data)
			case "prem":
				t.premultipliedBy = binary.BigEndian.Uint32(ref.data)
			default:
				return nil, errors.New("unsupported AVIF track reference")
			}
		}
	}
	if metadata, ok := f["meta"]; ok {
		if err := avifSequenceMeta(metadata, budget); err != nil {
			return nil, err
		}
	}
	mdia, err := avifSequenceFields(f["mdia"], budget)
	if err != nil {
		return nil, err
	}
	handler := mdia["hdlr"].data
	if len(handler) < 24 || (string(handler[8:12]) != "pict" && string(handler[8:12]) != "vide" && string(handler[8:12]) != "auxv") {
		return nil, errors.New("unsupported AVIF sequence handler")
	}
	mdhd := mdia["mdhd"].data
	var scale uint32
	if len(mdhd) == 24 && mdhd[0] == 0 {
		scale = binary.BigEndian.Uint32(mdhd[12:16])
	} else if len(mdhd) == 36 && mdhd[0] == 1 {
		scale = binary.BigEndian.Uint32(mdhd[20:24])
	} else {
		return nil, errors.New("invalid AVIF media header")
	}
	if scale == 0 {
		return nil, errors.New("AVIF sequence has zero timescale")
	}
	minf, err := avifSequenceFields(mdia["minf"], budget)
	if err != nil {
		return nil, err
	}
	if dinf, ok := minf["dinf"]; ok {
		if err := avifSequenceDataReference(dinf, budget); err != nil {
			return nil, err
		}
	}
	stbl, err := avifSequenceFields(minf["stbl"], budget)
	if err != nil {
		return nil, err
	}
	sizes, err := avifSequenceSampleSizes(stbl["stsz"].data)
	if err != nil {
		return nil, err
	}
	t.count = len(sizes)
	if int64(t.count) > MaxPixels/pixels {
		return nil, errors.New("AVIF sequence exceeds the cumulative canvas budget")
	}
	t.delays, err = avifSequenceDelays(stbl["stts"].data, scale, t.count)
	if err != nil {
		return nil, err
	}
	if err := avifSequenceExtents(stbl, sizes, media); err != nil {
		return nil, err
	}
	t.orientation, err = avifSequenceDescription(stbl["stsd"], t.width, t.height, t.auxiliaryFor != 0, budget)
	if err != nil {
		return nil, err
	}
	if sync, ok := stbl["stss"]; ok {
		if len(sync.data) < 8 || int(binary.BigEndian.Uint32(sync.data[4:8])) > t.count || len(sync.data) != 8+4*int(binary.BigEndian.Uint32(sync.data[4:8])) {
			return nil, errors.New("invalid AVIF sync sample table")
		}
	}
	return t, nil
}

func avifSequenceLoops(edits avifBox, duration uint64, budget *int) (int, error) {
	// Chromium maps absent edit lists (unknown repetitions) to infinite.
	if edits.kind == "" {
		return 0, nil
	}
	fields, err := avifSequenceFields(edits, budget)
	if err != nil {
		return 0, err
	}
	e := fields["elst"].data
	if len(e) < 4 || e[0] > 1 {
		return 0, errors.New("invalid AVIF edit list")
	}
	flags := uint32(e[1])<<16 | uint32(e[2])<<8 | uint32(e[3])
	if flags&1 == 0 {
		return -1, nil
	}
	if len(e) < 8 || binary.BigEndian.Uint32(e[4:8]) != 1 {
		return 0, errors.New("unsupported AVIF repeating edit list")
	}
	var segment uint64
	if e[0] == 1 {
		if len(e) != 28 {
			return 0, errors.New("invalid AVIF 64-bit edit list")
		}
		segment = binary.BigEndian.Uint64(e[8:16])
	} else {
		if len(e) != 20 {
			return 0, errors.New("invalid AVIF edit list")
		}
		segment = uint64(binary.BigEndian.Uint32(e[8:12]))
	}
	if segment == 0 || duration == 0 {
		return 0, errors.New("invalid AVIF repeating duration")
	}
	if duration == math.MaxUint64 {
		return 0, nil
	}
	cycles := duration / segment
	if duration%segment != 0 {
		cycles++
	}
	if cycles-1 > math.MaxInt32 {
		return 0, nil
	}
	if cycles == 1 {
		return -1, nil
	}
	return int(cycles - 1), nil
}

func avifSequenceDataReference(box avifBox, budget *int) error {
	f, err := avifSequenceFields(box, budget)
	if err != nil {
		return err
	}
	d := f["dref"]
	if len(d.data) < 8 || binary.BigEndian.Uint32(d.data[:4]) != 0 || binary.BigEndian.Uint32(d.data[4:8]) != 1 {
		return errors.New("unsupported AVIF data references")
	}
	refs, err := avifBoxes(d.data[8:], d.start+8, budget)
	if err != nil {
		return err
	}
	if len(refs) != 1 || refs[0].kind != "url " || len(refs[0].data) != 4 || binary.BigEndian.Uint32(refs[0].data) != 1 {
		return errors.New("external AVIF sequence data is unsupported")
	}
	return nil
}

func avifSequenceSampleSizes(data []byte) ([]uint32, error) {
	if len(data) < 12 || binary.BigEndian.Uint32(data[:4]) != 0 {
		return nil, errors.New("invalid AVIF sample sizes")
	}
	count := uint64(binary.BigEndian.Uint32(data[8:12]))
	uniform := binary.BigEndian.Uint32(data[4:8])
	if count == 0 || count > MaxFrames || (uniform == 0 && uint64(len(data)) != 12+4*count) || (uniform != 0 && len(data) != 12) {
		return nil, errors.New("AVIF sample count exceeds the frame limit or table bounds")
	}
	sizes := make([]uint32, int(count))
	for i := range sizes {
		sizes[i] = uniform
		if uniform == 0 {
			sizes[i] = binary.BigEndian.Uint32(data[12+4*i:])
		}
		if sizes[i] == 0 || uint64(sizes[i]) > uint64(MaxBytes) {
			return nil, errors.New("invalid AVIF sample byte size")
		}
	}
	return sizes, nil
}

func avifSequenceDelays(data []byte, scale uint32, count int) ([]time.Duration, error) {
	if len(data) < 8 || binary.BigEndian.Uint32(data[:4]) != 0 {
		return nil, errors.New("invalid AVIF sample timing")
	}
	n := uint64(binary.BigEndian.Uint32(data[4:8]))
	if n == 0 || n > uint64(count) || uint64(len(data)) != 8+8*n {
		return nil, errors.New("invalid AVIF timing table bounds")
	}
	delays := make([]time.Duration, 0, count)
	var total time.Duration
	for p := 8; p < len(data); p += 8 {
		frames := uint64(binary.BigEndian.Uint32(data[p : p+4]))
		delta := uint64(binary.BigEndian.Uint32(data[p+4 : p+8]))
		if frames == 0 || frames > uint64(count-len(delays)) {
			return nil, errors.New("AVIF timing sample count mismatch")
		}
		// uint32 * 1e9 fits uint64. Normalize the same small delays as browser
		// image animation, before checking aggregate time.Duration overflow.
		delay := time.Duration(delta * uint64(time.Second) / uint64(scale))
		if delay <= 10*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		if uint64(delay) > uint64(math.MaxInt64-total)/frames {
			return nil, errors.New("AVIF animation duration exceeds timing bounds")
		}
		total += delay * time.Duration(frames)
		for range int(frames) {
			delays = append(delays, delay)
		}
	}
	if len(delays) != count {
		return nil, errors.New("incomplete AVIF timing samples")
	}
	return delays, nil
}

func avifSequenceExtents(fields map[string]avifBox, sizes []uint32, media []avifBox) error {
	offsets := fields["stco"].data
	step := 4
	if other, ok := fields["co64"]; ok {
		if len(offsets) != 0 {
			return errors.New("duplicate AVIF chunk offset tables")
		}
		offsets = other.data
		step = 8
	}
	if len(offsets) < 8 || binary.BigEndian.Uint32(offsets[:4]) != 0 {
		return errors.New("invalid AVIF chunk offsets")
	}
	n := uint64(binary.BigEndian.Uint32(offsets[4:8]))
	if n == 0 || n > uint64(len(sizes)) || uint64(len(offsets)) != 8+n*uint64(step) {
		return errors.New("invalid AVIF chunk count")
	}
	mapping := fields["stsc"].data
	if len(mapping) < 8 || binary.BigEndian.Uint32(mapping[:4]) != 0 {
		return errors.New("invalid AVIF sample/chunk mapping")
	}
	m := uint64(binary.BigEndian.Uint32(mapping[4:8]))
	if m == 0 || m > n || uint64(len(mapping)) != 8+12*m {
		return errors.New("invalid AVIF sample/chunk table")
	}
	first := func(i int) uint32 { return binary.BigEndian.Uint32(mapping[8+12*i:]) }
	for i := 0; i < int(m); i++ {
		if (i == 0 && first(i) != 1) || (i > 0 && first(i) <= first(i-1)) || uint64(first(i)) > n || binary.BigEndian.Uint32(mapping[16+12*i:]) != 1 {
			return errors.New("unsupported AVIF sample description mapping")
		}
	}
	sample, group := 0, 0
	for chunk := 0; chunk < int(n); chunk++ {
		if group+1 < int(m) && uint32(chunk+1) >= first(group+1) {
			group++
		}
		per := uint64(binary.BigEndian.Uint32(mapping[12+12*group:]))
		if per == 0 || per > uint64(len(sizes)-sample) {
			return errors.New("invalid AVIF samples per chunk")
		}
		p := 8 + chunk*step
		offset := uint64(binary.BigEndian.Uint32(offsets[p : p+4]))
		if step == 8 {
			offset = binary.BigEndian.Uint64(offsets[p : p+8])
		}
		for range int(per) {
			size := uint64(sizes[sample])
			inside := false
			for _, box := range media {
				if offset >= box.start && offset <= box.end && size <= box.end-offset {
					inside = true
					break
				}
			}
			if !inside {
				return errors.New("AVIF sample extent is outside mdat")
			}
			offset += size
			sample++
		}
	}
	if sample != len(sizes) {
		return errors.New("incomplete AVIF sample/chunk mapping")
	}
	return nil
}

func avifSequenceDescription(box avifBox, width, height int, alpha bool, budget *int) (int, error) {
	data := box.data
	if len(data) < 8 || data[0] > 1 || !avifZero(data[1:4]) || binary.BigEndian.Uint32(data[4:8]) != 1 {
		return 0, errors.New("unsupported AVIF sequence descriptions")
	}
	entries, err := avifBoxes(data[8:], box.start+8, budget)
	if err != nil {
		return 0, err
	}
	if len(entries) != 1 || entries[0].kind != "av01" || len(entries[0].data) < 78 {
		return 0, errors.New("AVIF sequence is not one AV1 description")
	}
	e := entries[0]
	if binary.BigEndian.Uint16(e.data[6:8]) != 1 || int(binary.BigEndian.Uint16(e.data[24:26])) != width || int(binary.BigEndian.Uint16(e.data[26:28])) != height {
		return 0, errors.New("AVIF description dimensions or resource reference differ")
	}
	properties, err := avifBoxes(e.data[78:], e.start+78, budget)
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	config := false
	rot, mir := -1, -1
	for _, p := range properties {
		if seen[p.kind] {
			return 0, errors.New("duplicate AVIF sequence image property")
		}
		seen[p.kind] = true
		switch p.kind {
		case "av1C":
			if len(p.data) != 4 || p.data[0] != 0x81 || p.data[1]>>5 > 2 || p.data[3] != 0 {
				return 0, errors.New("unsupported AVIF codec configuration")
			}
			config = true
		case "colr":
			if len(p.data) != 11 || string(p.data[:4]) != "nclx" || p.data[10]&0x7f != 0 {
				return 0, errors.New("unsupported AVIF color property")
			}
		case "pasp":
			if len(p.data) != 8 || binary.BigEndian.Uint32(p.data[:4]) == 0 || binary.BigEndian.Uint32(p.data[4:]) == 0 {
				return 0, errors.New("invalid AVIF aspect property")
			}
		case "ispe":
			if len(p.data) != 12 || binary.BigEndian.Uint32(p.data[:4]) != 0 || int(binary.BigEndian.Uint32(p.data[4:8])) != width || int(binary.BigEndian.Uint32(p.data[8:12])) != height {
				return 0, errors.New("AVIF track spatial extents differ")
			}
		case "pixi":
			if len(p.data) < 5 || binary.BigEndian.Uint32(p.data[:4]) != 0 || (p.data[4] != 1 && p.data[4] != 3) || len(p.data) != 5+int(p.data[4]) {
				return 0, errors.New("invalid AVIF track pixel information")
			}
			for _, depth := range p.data[5:] {
				if depth != 8 && depth != 10 && depth != 12 {
					return 0, errors.New("unsupported AVIF track pixel depth")
				}
			}
		case "irot":
			if len(p.data) != 1 || p.data[0] > 3 {
				return 0, errors.New("invalid AVIF rotation")
			}
			rot = int(p.data[0])
		case "imir":
			if len(p.data) != 1 || p.data[0] > 1 {
				return 0, errors.New("invalid AVIF mirror")
			}
			mir = int(p.data[0])
		case "auxi":
			if !alpha || len(p.data) < 5 || binary.BigEndian.Uint32(p.data[:4]) != 0 || string(p.data[4:]) != "urn:mpeg:mpegB:cicp:systems:auxiliary:alpha\x00" {
				return 0, errors.New("unsupported AVIF auxiliary sequence")
			}
		case "clli":
			if len(p.data) != 4 {
				return 0, errors.New("invalid AVIF light property")
			}
		case "mdcv":
			if len(p.data) != 24 {
				return 0, errors.New("invalid AVIF mastering property")
			}
		default:
			return 0, errors.New("unsupported AVIF sequence image property")
		}
	}
	if !config {
		return 0, errors.New("missing AVIF sequence codec configuration")
	}
	return exifOrientationFromIrotImir(rot >= 0, rot, mir >= 0, mir), nil
}
