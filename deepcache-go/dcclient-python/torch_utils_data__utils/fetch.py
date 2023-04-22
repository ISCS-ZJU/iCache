r""""Contains definitions of the methods used by the _BaseDataLoaderIter to fetch
data from an iterable-style or map-style dataset. This logic is shared in both
single- and multi-processing data loading.
"""


import os, time

class _BaseDatasetFetcher(object):
    def __init__(self, dataset, auto_collation, collate_fn, drop_last):
        self.dataset = dataset
        self.auto_collation = auto_collation
        self.collate_fn = collate_fn
        self.drop_last = drop_last

    def fetch(self, possibly_batched_index):
        raise NotImplementedError()


class _IterableDatasetFetcher(_BaseDatasetFetcher):
    def __init__(self, dataset, auto_collation, collate_fn, drop_last):
        super(_IterableDatasetFetcher, self).__init__(dataset, auto_collation, collate_fn, drop_last)
        self.dataset_iter = iter(dataset)

    def fetch(self, possibly_batched_index):
        if self.auto_collation:
            data = []
            for _ in possibly_batched_index:
                try:
                    data.append(next(self.dataset_iter))
                except StopIteration:
                    break
            if len(data) == 0 or (self.drop_last and len(data) < len(possibly_batched_index)):
                raise StopIteration
        else:
            data = next(self.dataset_iter)
        return self.collate_fn(data)


class _MapDatasetFetcher(_BaseDatasetFetcher):
    def __init__(self, dataset, auto_collation, collate_fn, drop_last, cur_epoch, warm_up, reuse_factor):
        self.dataset = dataset
        super(_MapDatasetFetcher, self).__init__(dataset, auto_collation, collate_fn, drop_last)
        # tell if all fetchers in this epoch should fetch all data
        self.full_access = True if cur_epoch < warm_up or cur_epoch % reuse_factor==0 else False
        # self.full_access = False

    def fetch(self, possibly_batched_index, validation=False, stub=None, fakecache=False):
        worker_pid = os.getpid()
        
        ## start = time.time()
        data = []
        if self.auto_collation:
            for idx in possibly_batched_index:
                if type(idx) == int:
                    # rt = self.dataset[idx]
                    rt = self.dataset.__getitem__(idx, stub, fakecache)
                else:
                    # idx is an tuple (imgidx, 1 or 0) denotes whether important
                    rt = self.dataset.getitem(idx, stub, self.full_access, fakecache)
                if validation == False:
                    while (rt[-2] == -1):
                        if type(idx) == int:
                            rt = self.dataset.__getitem__(idx, stub, fakecache)
                        else:
                            # idx is an tuple (imgidx, 1 or 0) denotes whether important
                            rt = self.dataset.getitem(idx, stub, self.full_access, fakecache)
                data.append(rt)
            #data = [self.dataset[idx] for idx in possibly_batched_index]
        else:
            data = self.dataset[possibly_batched_index]
        return self.collate_fn(data)

class _MapDatasetQuiverFetcher(_BaseDatasetFetcher):
    """for quiver fetch batch data"""
    def __init__(self, dataset, auto_collation, collate_fn, drop_last, batch_size):
        self.batch_size = batch_size
        self.dataset = dataset
        super(_MapDatasetQuiverFetcher, self).__init__(dataset, auto_collation, collate_fn, drop_last)


    def quiver_fetch(self, possibly_batched_index, stub, fakecache):
        worker_pid = os.getpid()
        remain_unused_id_list = []
        ## start = time.time()
        data = []
        if self.auto_collation:
            cnt = 0
            for idx in possibly_batched_index:
                
                rt = self.dataset.__getitem__(idx, stub, fakecache)
                
                cnt += 1
                if rt[-2] != -1: # if not exists, server return imgidx==-1
                    data.append(rt)
                    # if already got batch size data
                    if len(data) == self.batch_size:
                        # for idx in possibly_batched_index[-1:cnt-1:-1]:
                        #     global_indices.append(idx)
                        remain_unused_id_list.extend(possibly_batched_index[cnt:])
                        break
                else:
                    remain_unused_id_list.append(idx)
            # print("hit data len:", len(data))
            lack_n = min(self.batch_size, len(possibly_batched_index))- len(data)
            if lack_n > 0:
                for i in range(lack_n):
                    data.append(self.dataset.__getitem__(-1, stub, fakecache)) # request == -1 means substitutebility
            remain_unused_id_list = remain_unused_id_list[lack_n:]
            # print("len(global_indices) after quiver access fetch:", len(global_indices))
        else:
            assert (1>2) # panic
            data = self.dataset[possibly_batched_index]
        ## end = time.time()
        ## print(f"worker_pid: {worker_pid}; batch_fetch_time: {end-start}")
        
        # start = time.time()
        return self.collate_fn(data), remain_unused_id_list