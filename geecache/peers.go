package geecache

import pb "geecache/geecachepb"

// PeerPicker 是必须实现的接口，用于定位拥有特定键的对等点。
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是对等点必须实现的接口。
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
