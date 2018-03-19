package dmx

import (
	"fmt"

	"github.com/StageAutoControl/controller/cntl"
)

// StreamlineScenes returns a map of frame -> scene that is easier to handle then a plain array
func StreamlineScenes(ds *cntl.DataStore, s *cntl.Song) (map[uint64][]*cntl.DMXScene, error) {
	scs := make(map[uint64][]*cntl.DMXScene)
	for _, sp := range s.DMXScenes {
		sc, ok := ds.DMXScenes[sp.ID]
		if !ok {
			return map[uint64][]*cntl.DMXScene{}, fmt.Errorf("cannot find DMXScene %q", sp.ID)
		}

		l := CalcSceneLength(sc)
		at := uint64(sp.At)

		for i := uint64(0); i <= uint64(sp.Repeat); i++ {
			pos := at + i*l
			if _, ok := scs[pos]; !ok {
				scs[pos] = make([]*cntl.DMXScene, 0)
			}

			scs[pos] = append(scs[pos], sc)
		}
	}

	return scs, nil
}

// CalcSceneLength calculates the length of a given scene in render frames
func CalcSceneLength(sc *cntl.DMXScene) uint64 {
	return uint64(sc.NoteCount * (cntl.RenderFrames / sc.NoteValue))
}

// RenderScene renders the given dmx scene to dmx commands and returns them.
// The first array dimension contains the render frames, the second dimension contains all
// dmx commands for a render frame.
func RenderScene(ds *cntl.DataStore, sc *cntl.DMXScene) ([]cntl.DMXCommands, error) {
	sceneLength := uint8(CalcSceneLength(sc))
	cmds := make([]cntl.DMXCommands, sceneLength)

	for i, ss := range sc.SubScenes {
		var scs []cntl.DMXCommands

		if len(ss.DeviceParams) > 0 && ss.Preset != "" {
			return []cntl.DMXCommands{}, fmt.Errorf("SubScene %d of scene %q cannot have both params and a preset", i, sc.ID)
		}

		if ss.Preset != "" {
			p, ok := ds.DMXPresets[ss.Preset]
			if !ok {
				return []cntl.DMXCommands{}, fmt.Errorf("cannot find DMXPreset %q", ss.Preset)
			}

			pcs, err := RenderPreset(ds, p)
			if err != nil {
				return []cntl.DMXCommands{}, err
			}

			scs = Merge(scs, pcs)
		}

		for _, dp := range ss.DeviceParams {
			dcs, err := RenderDeviceParams(ds, &dp)
			if err != nil {
				return []cntl.DMXCommands{}, fmt.Errorf("failed to render scene %q: %v", sc.ID, err)
			}

			scs = Merge(scs, dcs)
		}

		for _, at := range ss.At {
			pos := uint64(sceneLength/sc.NoteCount) * at
			cmds = MergeAtOffset(cmds, scs, int(pos))
		}
	}

	return cmds, nil
}