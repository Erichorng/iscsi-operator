package resource

import (
	"context"

	rook "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildPool(ctx context.Context, name string, ns string, ps rook.PoolSpec) *rook.CephBlockPool {

	cephblockpool := &rook.CephBlockPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: rook.NamedBlockPoolSpec{
			PoolSpec: ps,
		},
	}
	return cephblockpool

}
