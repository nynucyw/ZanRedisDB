package cluster

import (
	"errors"
	"strconv"
)

var (
	ErrKeyAlreadyExist = errors.New("Key already exist")
	ErrKeyNotFound     = errors.New("Key not found")
)

type EpochType int64

type NodeInfo struct {
	ID       string
	NodeIP   string
	TcpPort  string
	HttpPort string
	RpcPort  string
	Epoch    EpochType
}

func (self *NodeInfo) GetID() string {
	return self.ID
}

type NamespaceMetaInfo struct {
	PartitionNum int
	Replica      int
	// to verify the data of the create -> delete -> create with same namespace
	MagicCode int64
	MetaEpoch EpochType
}

type PartitionReplicaInfo struct {
	RaftNodes []string
	MaxRaftID int64
	RaftIDs   map[string]int64
	Epoch     EpochType
}

func GetDesp(ns string, part int) string {
	return ns + "-" + strconv.Itoa(part)
}

type PartitionMetaInfo struct {
	Name      string
	Partition int
	NamespaceMetaInfo
	PartitionReplicaInfo
}

func (self *PartitionMetaInfo) GetDesp() string {
	return self.Name + "-" + strconv.Itoa(self.Partition)
}

type ConsistentStore interface {
	WriteKey(key, value string) error
	ReadKey(key string) (string, error)
	ListKey(key string) ([]string, error)
}

type Register interface {
	InitClusterID(id string)
	// all registered pd nodes.
	GetAllPDNodes() ([]NodeInfo, error)
	// should return both the meta info for namespace and the replica info for partition
	// epoch should be updated while return
	GetNamespacePartInfo(ns string, partition int) (*PartitionMetaInfo, error)
	// get  meta info only
	GetNamespaceMetaInfo(ns string) (NamespaceMetaInfo, error)
	GetNamespaceInfo(ns string) ([]PartitionMetaInfo, error)
	GetAllNamespaces() (map[string][]PartitionMetaInfo, error)
	GetNamespacesNotifyChan() chan struct{}
}

// We need check leader before do any modify to etcd.
// Make sure all returned value should be copied to avoid modify by outside.
type PDRegister interface {
	Register
	Register(nodeData *NodeInfo) error // update
	Unregister(nodeData *NodeInfo) error
	Stop()
	// the cluster root modify index
	GetClusterEpoch() (EpochType, error)
	AcquireAndWatchLeader(leader chan *NodeInfo, stop chan struct{})

	GetDataNodes() ([]NodeInfo, error)
	// watching the cluster data node, should return the newest for the first time.
	WatchDataNodes(nodeC chan []NodeInfo, stopC chan struct{})
	// create and write the meta info to meta node
	CreateNamespace(ns string, meta *NamespaceMetaInfo) error
	// create partition path
	CreateNamespacePartition(ns string, partition int) error
	IsExistNamespace(ns string) (bool, error)
	IsExistNamespacePartition(ns string, partition int) (bool, error)
	DeleteNamespacePart(ns string, partition int) error
	DeleteWholeNamespace(ns string) error
	//
	// update the replica info about replica node list, epoch for partition
	// Note: update should do check-and-set to avoid unexpected override.
	// the epoch in replicaInfo should be updated to the new epoch
	// if no partition, replica info node should create only once.
	UpdateNamespacePartReplicaInfo(ns string, partition int, replicaInfo *PartitionReplicaInfo, oldGen EpochType) error
}

type DataNodeRegister interface {
	Register
	Register(nodeData *NodeInfo) error // update
	Unregister(nodeData *NodeInfo) error
	// get the newest pd leader and watch the change of it.
	WatchPDLeader(leader chan *NodeInfo, stop chan struct{}) error
}