// Copyright (c) 2017 OpenSDS Authors.
//
//    Licensed under the Apache License, Version 2.0 (the "License"); you may
//    not use this file except in compliance with the License. You may obtain
//    a copy of the License at
//
//         http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//    License for the specific language governing permissions and limitations
//    under the License.

package ceph

import (
	"errors"
	"testing"
	"unsafe"

	"strings"

	"github.com/bouk/monkey"
	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	pb "github.com/opensds/opensds/pkg/dock/proto"
	"github.com/opensds/opensds/pkg/utils/config"
	"github.com/satori/go.uuid"
)

func TestCreateVolume(t *testing.T) {

	defer monkey.UnpatchAll()
	monkey.Patch((*Driver).initConn, func(d *Driver) error { return nil })
	monkey.Patch(rbd.Create, func(ioctx *rados.IOContext, name string, size uint64, order int,
		args ...uint64) (*rbd.Image, error) {
		return nil, nil
	})
	monkey.Patch((*rados.Conn).Shutdown, func(c *rados.Conn) {})
	monkey.Patch((*rados.IOContext).Destroy, func(ioctx *rados.IOContext) {})

	// case 1
	d := Driver{}
	resp, err := d.CreateVolume(&pb.CreateVolumeOpts{Name: "volume001", Size: 1})
	if err != nil {
		t.Errorf("Test Create volume error")
	}
	if resp.Size != 1 {
		t.Errorf("Test Create volume size error")
	}
	if resp.Name != "volume001" {
		t.Errorf("Test Create volume name error")
	}
	if _, err = uuid.FromString(resp.Id); err != nil {
		t.Errorf("Test Create volume uuid error")
	}

	//case 2
	monkey.Unpatch((*Driver).initConn)
	monkey.Patch((*Driver).initConn, func(d *Driver) error {
		return errors.New("Fake error")
	})
	d = Driver{}
	_, err = d.CreateVolume(&pb.CreateVolumeOpts{Name: "volume001", Size: 1})
	if err == nil {
		t.Errorf("Test Create volume error")
	}

	//case 3
	monkey.Unpatch(rbd.Create)
	monkey.Patch(rbd.Create, func(ioctx *rados.IOContext, name string, size uint64, order int,
		args ...uint64) (*rbd.Image, error) {
		return nil, errors.New("Fake error")
	})
	_, err = d.CreateVolume(&pb.CreateVolumeOpts{Name: "volume001", Size: 1})
	if err == nil {
		t.Errorf("Test Create volume error")
	}
}

func TestGetVolume(t *testing.T) {
	defer monkey.UnpatchAll()
	monkey.Patch((*Driver).initConn, func(d *Driver) error {
		return nil
	})
	monkey.Patch(rbd.GetImageNames, func(ioctx *rados.IOContext) (names []string, err error) {
		nameList := []string{opensdsPrefix + ":volume001:7ee11866-1f40-4f3c-b093-7a3684523a19"}
		return nameList, nil
	})
	monkey.Patch((*rbd.Image).GetSize, func(r *rbd.Image) (size uint64, err error) {
		return 1 << sizeShiftBit, nil
	})
	monkey.Patch((*rbd.Image).Open, func(r *rbd.Image, args ...interface{}) error {
		return nil
	})
	monkey.Patch((*rbd.Image).Close, func(r *rbd.Image) error {
		return nil
	})
	monkey.Patch((*rados.Conn).Shutdown, func(c *rados.Conn) {})
	monkey.Patch((*rados.IOContext).Destroy, func(ioctx *rados.IOContext) {})

	// case 1
	d := Driver{}
	resp, err := d.PullVolume("7ee11866-1f40-4f3c-b093-7a3684523a19")
	if err != nil {
		t.Errorf("Test Get volume error")
	}
	if resp.Size != 1 {
		t.Errorf("Test Get volume size error")
	}
	if resp.Name != "volume001" {
		t.Errorf("Test Get volume name error")
	}
	if resp.Id != "7ee11866-1f40-4f3c-b093-7a3684523a19" {
		t.Errorf("Test Get volume uuid error")
	}

	resp, err = d.PullVolume("11111111-1111-1111-1111-111111111111")
	if err != rbd.RbdErrorNotFound {
		t.Errorf("Test Get volume error")
	}
}

func TestDeleteVolme(t *testing.T) {
	defer monkey.UnpatchAll()
	monkey.Patch((*Driver).initConn, func(d *Driver) error {
		return nil
	})
	monkey.Patch(rbd.GetImageNames, func(ioctx *rados.IOContext) (names []string, err error) {
		nameList := []string{opensdsPrefix + ":volume001:7ee11866-1f40-4f3c-b093-7a3684523a19"}
		return nameList, nil
	})

	monkey.Patch((*rbd.Image).GetSize, func(r *rbd.Image) (size uint64, err error) {
		return 1 << sizeShiftBit, nil
	})
	monkey.Patch((*rbd.Image).Remove, func(r *rbd.Image) error {
		return nil
	})
	monkey.Patch((*rados.Conn).Shutdown, func(c *rados.Conn) {})
	monkey.Patch((*rados.IOContext).Destroy, func(ioctx *rados.IOContext) {})

	// case 1
	d := Driver{}
	opt := &pb.DeleteVolumeOpts{Id: "7ee11866-1f40-4f3c-b093-7a3684523a19"}
	err := d.DeleteVolume(opt)
	if err != nil {
		t.Errorf("Test Delete volume error")
	}
}

func TestCreateSnapshot(t *testing.T) {
	defer monkey.UnpatchAll()
	monkey.Patch((*Driver).initConn, func(d *Driver) error {
		return nil
	})
	monkey.Patch(rbd.GetImageNames, func(ioctx *rados.IOContext) (names []string, err error) {
		nameList := []string{opensdsPrefix + ":volume001:7ee11866-1f40-4f3c-b093-7a3684523a19"}
		return nameList, nil
	})

	monkey.Patch((*rbd.Image).GetSize, func(r *rbd.Image) (size uint64, err error) {
		return 1 << sizeShiftBit, nil
	})
	//
	type Snapshot struct {
		image *rbd.Image
		name  string
	}
	monkey.Patch((*rbd.Image).CreateSnapshot, func(image *rbd.Image, snapname string) (*rbd.Snapshot, error) {
		snapshot := &rbd.Snapshot{}
		p := (*Snapshot)(unsafe.Pointer(snapshot))
		p.name = "snapshot001"
		p.image = nil
		return snapshot, nil
	})

	monkey.Patch((*rbd.Image).Open, func(r *rbd.Image, args ...interface{}) error { return nil })
	monkey.Patch((*rbd.Image).Close, func(r *rbd.Image) error { return nil })
	monkey.Patch((*rados.Conn).Shutdown, func(c *rados.Conn) {})
	monkey.Patch((*rados.IOContext).Destroy, func(ioctx *rados.IOContext) {})

	// case 1
	d := Driver{}
	resp, err := d.CreateSnapshot(&pb.CreateVolumeSnapshotOpts{
		Name:        "snapshot001",
		Id:          "7ee11866-1f40-4f3c-b093-7a3684523a19",
		Description: "unite test"})

	if err != nil {
		t.Errorf("Test Create snapshot error")
	}
	if resp.Name != "snapshot001" {
		t.Errorf("Test Create snapshot name error")
	}
	if resp.VolumeId != "7ee11866-1f40-4f3c-b093-7a3684523a19" {
		t.Errorf("Test Create snapshot name error")
	}
	if _, err = uuid.FromString(resp.Id); err != nil {
		t.Errorf("Test Create snapshot error")
	}
}

func TestGetSnapshot(t *testing.T) {
	defer monkey.UnpatchAll()
	monkey.Patch((*Driver).initConn, func(d *Driver) error {
		return nil
	})
	monkey.Patch(rbd.GetImageNames, func(ioctx *rados.IOContext) (names []string, err error) {
		nameList := []string{opensdsPrefix + ":volume001:7ee11866-1f40-4f3c-b093-7a3684523a19"}
		return nameList, nil
	})
	monkey.Patch((*rbd.Image).GetSnapshotNames, func(*rbd.Image) (snaps []rbd.SnapInfo, err error) {
		snaps = make([]rbd.SnapInfo, 1)
		snaps[0] = rbd.SnapInfo{Id: uint64(1),
			Size: uint64(1 << sizeShiftBit),
			Name: opensdsPrefix + ":snapshot001:25f5d7a2-553d-4d6c-904d-179a9e698cf8",
		}
		return snaps, nil
	})
	monkey.Patch((*rbd.Image).GetSize, func(r *rbd.Image) (size uint64, err error) {
		return 1 << sizeShiftBit, nil
	})

	type Snapshot struct {
		image *rbd.Image
		name  string
	}
	monkey.Patch((*rbd.Image).CreateSnapshot, func(image *rbd.Image, snapname string) (*rbd.Snapshot, error) {
		snapshot := &rbd.Snapshot{}
		p := (*Snapshot)(unsafe.Pointer(snapshot))
		p.name = snapname
		p.image = image
		return snapshot, nil
	})

	monkey.Patch((*rbd.Image).Open, func(r *rbd.Image, args ...interface{}) error { return nil })
	monkey.Patch((*rbd.Image).Close, func(r *rbd.Image) error { return nil })
	monkey.Patch((*rados.Conn).Shutdown, func(c *rados.Conn) {})
	monkey.Patch((*rados.IOContext).Destroy, func(ioctx *rados.IOContext) {})

	// case 1
	d := Driver{}
	resp, err := d.PullSnapshot("25f5d7a2-553d-4d6c-904d-179a9e698cf8")
	if err != nil {
		t.Errorf("Test Get snapshot error")
	}
	if resp.Name != "snapshot001" {
		t.Errorf("Test Get snapshot name error")
	}
	if resp.Size != 1 {
		t.Errorf("Test Get snapshot size error")
	}

	// case 2
	_, err = d.PullSnapshot("11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Errorf("Test Get snapshot error")
	}
}

func TestDeleteSnapshot(t *testing.T) {
	defer monkey.UnpatchAll()
	monkey.Patch((*Driver).initConn, func(d *Driver) error {
		return nil
	})
	monkey.Patch(rbd.GetImageNames, func(ioctx *rados.IOContext) (names []string, err error) {
		nameList := []string{opensdsPrefix + ":volume001:7ee11866-1f40-4f3c-b093-7a3684523a19"}
		return nameList, nil
	})
	monkey.Patch((*rbd.Image).GetSnapshotNames, func(*rbd.Image) (snaps []rbd.SnapInfo, err error) {
		snaps = make([]rbd.SnapInfo, 1)
		snaps[0] = rbd.SnapInfo{Id: uint64(1),
			Size: uint64(1 << sizeShiftBit),
			Name: opensdsPrefix + ":snapshot001:25f5d7a2-553d-4d6c-904d-179a9e698cf8",
		}
		return snaps, nil
	})
	monkey.Patch((*rbd.Image).GetSize, func(r *rbd.Image) (size uint64, err error) {
		return 1 << sizeShiftBit, nil
	})

	type Snapshot struct {
		image *rbd.Image
		name  string
	}
	monkey.Patch((*rbd.Snapshot).Remove, func(*rbd.Snapshot) error {
		return nil
	})

	monkey.Patch((*rbd.Image).Open, func(r *rbd.Image, args ...interface{}) error { return nil })
	monkey.Patch((*rbd.Image).Close, func(r *rbd.Image) error { return nil })
	monkey.Patch((*rados.Conn).Shutdown, func(c *rados.Conn) {})
	monkey.Patch((*rados.IOContext).Destroy, func(ioctx *rados.IOContext) {})

	// case 1
	d := Driver{}
	err := d.DeleteSnapshot(&pb.DeleteVolumeSnapshotOpts{Id: "25f5d7a2-553d-4d6c-904d-179a9e698cf8"})
	if err != nil {
		t.Errorf("Test Delete snapshot error")
	}
}

func TestCephConfig(t *testing.T) {
	config.CONF.OsdsDock.CephConfig = "testdata/ceph.yaml"
	conf := getConfig()
	if conf.ConfigFile != "/etc/ceph/ceph.conf" {
		t.Error("Test ConfigFile failed!")
	}
	if conf.Pool["rbd"].DiskType != "SSD" {
		t.Error("Test ConfigFile DiskType failed!")
	}
	if conf.Pool["rbd"].IOPS != 1000 {
		t.Error("Test ConfigFile IOPS failed!")
	}
	if conf.Pool["rbd"].BandWidth != 1000 {
		t.Error("Test ConfigFile BandWidth failed!")
	}
	if conf.Pool["test"].DiskType != "SAS" {
		t.Error("Test ConfigFile DiskType failed!")
	}
	if conf.Pool["test"].IOPS != 800 {
		t.Error("Test ConfigFile IOPS failed!")
	}
	if conf.Pool["test"].BandWidth != 800 {
		t.Error("Test ConfigFile BandWidth failed!")
	}
}

func TestListPools(t *testing.T) {

	defer monkey.UnpatchAll()
	monkey.Patch(execCmd, func(cmd string) (string, error) {
		cephDuInfo := "" +
			"GLOBAL:\n" +
			"    SIZE       AVAIL     RAW USED     %RAW USED\n" +
			"    19053M     6859M       12194M         64.00\n" +
			"POOLS:\n" +
			"    NAME                ID     USED     %USED     MAX AVAIL     OBJECTS\n" +
			"    rbd                 0      942M     12.21         2286M         245\n" +
			"    test                1         0         0         2286M           1\n" +
			"    pool001             2         0         0         2286M           0\n" +
			"    testpoolerasure     3         0         0         4572M           0\n" +
			"    NAME                9         0         0         2286M           0\n" +
			"    ecpool              10        0         0         4115M           0\n" +
			"    12                  11        0         0         2286M           0"
		poolAttrInfo := "" +
			"'rbd' replicated 3 0\n" +
			"'test' replicated 3 0\n" +
			"'pool001' replicated 3 0\n" +
			"'testpoolerasure' erasure 3 1\n" +
			"'NAME' replicated 3 0\n" +
			"'ecpool' erasure 5 2\n" +
			"'12' replicated 3 0"
		if strings.HasPrefix(cmd, "ceph df") {
			return cephDuInfo, nil
		}
		return poolAttrInfo, nil
	})

	d := Driver{}
	pols, err := d.ListPools()
	if err != nil {
		t.Errorf("Test List Pools error")
	}
	if pols[0].Name != "rbd" {
		t.Errorf("Test List Pools Name error")
	}
	if pols[0].Id != "0517f561-85b3-5f6a-a38d-8b5a08bff7df" {
		t.Errorf("Test List Pools UUID error")
	}
	if pols[0].FreeCapacity != 2 {
		t.Errorf("Test List Pools FreeCapacity error")
	}

	if pols[0].TotalCapacity != 6 {
		t.Errorf("Test List Pools TotalCapacity error")
	}

	if pols[0].Parameters["redundancyType"] != "replicated" {
		t.Errorf("Test List Pools redundancyType error")
	}

	if pols[0].Parameters["replicateSize"] != "3" {
		t.Errorf("Test List Pools replicateSize error")
	}

	if pols[0].Parameters["crushRuleset"] != "0" {
		t.Errorf("Test List Pools crushRuleset error")
	}

	if pols[0].Parameters["diskType"] != "SSD" {
		t.Errorf("Test List Pools diskType error")
	}

	if pols[0].Parameters["iops"] != int64(1000) {
		t.Errorf("Test List Pools iops error")
	}

	if pols[0].Parameters["bandwidth"] != int64(1000) {
		t.Errorf("Test List Pools bandWidth error")
	}

	if pols[5].Name != "ecpool" {
		t.Errorf("Test List Pools Name error")
	}

	if pols[5].Parameters["redundancyType"] != "erasure" {
		t.Errorf("Test List Pools redundancyType error")
	}

	if pols[5].Parameters["erasureSize"] != "5" {
		t.Errorf("Test List Pools replicateSize error")
	}

	if pols[5].Parameters["crushRuleset"] != "2" {
		t.Errorf("Test List Pools crushRuleset error")
	}

	if len(pols) != 6 {
		t.Errorf("Test List Pools len error")
	}
}
