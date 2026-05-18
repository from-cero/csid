package registry

// type EtcdRegistry struct {
//     client  *clientv3.Client
//     leaseID clientv3.LeaseID
//     nodeID  int64
// }

// func (r *EtcdRegistry) Acquire(ctx context.Context) (int64, error) {
//     lease, err := r.client.Grant(ctx, 30) // 30s TTL
//     if err != nil {
//         return -1, err
//     }
//     r.leaseID = lease.ID

//     for id := int64(0); id <= 1023; id++ {
//         key := fmt.Sprintf("/snowflake/workers/%d", id)
//         txn := r.client.Txn(ctx).
//             If(clientv3.Compare(clientv3.Version(key), "=", 0)).
//             Then(clientv3.OpPut(key, "", clientv3.WithLease(lease.ID)))

//         resp, err := txn.Commit()
//         if err != nil {
//             return -1, err
//         }
//         if resp.Succeeded {
//             r.nodeID = id
//             return id, nil
//         }
//     }
//     return -1, fmt.Errorf("no available node IDs")
// }

// func (r *EtcdRegistry) Release(ctx context.Context) error {
//     _, err := r.client.Revoke(ctx, r.leaseID)
//     return err
// }

// func (r *EtcdRegistry) Renew(ctx context.Context) error {
//     _, err := r.client.KeepAliveOnce(ctx, r.leaseID)
//     return err
// }

// func (r *EtcdRegistry) NodeID() int64 { return r.nodeID }
