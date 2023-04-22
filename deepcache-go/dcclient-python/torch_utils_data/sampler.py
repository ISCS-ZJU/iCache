import torch
from torch._six import int_classes as _int_classes
from torch import Tensor

from typing import Iterator, Optional, Sequence, List, TypeVar, Generic, Sized

import math, random, collections, numpy as np, time

import torch.distributed as dist

T_co = TypeVar('T_co', covariant=True)

class Sampler(Generic[T_co]):
    r"""Base class for all Samplers.

    Every Sampler subclass has to provide an :meth:`__iter__` method, providing a
    way to iterate over indices of dataset elements, and a :meth:`__len__` method
    that returns the length of the returned iterators.

    .. note:: The :meth:`__len__` method isn't strictly required by
              :class:`~torch.utils.data.DataLoader`, but is expected in any
              calculation involving the length of a :class:`~torch.utils.data.DataLoader`.
    """

    def __init__(self, data_source: Optional[Sized]) -> None:
        pass

    def __iter__(self) -> Iterator[T_co]:
        raise NotImplementedError

    # NOTE [ Lack of Default `__len__` in Python Abstract Base Classes ]
    #
    # Many times we have an abstract class representing a collection/iterable of
    # data, e.g., `torch.utils.data.Sampler`, with its subclasses optionally
    # implementing a `__len__` method. In such cases, we must make sure to not
    # provide a default implementation, because both straightforward default
    # implementations have their issues:
    #
    #   + `return NotImplemented`:
    #     Calling `len(subclass_instance)` raises:
    #       TypeError: 'NotImplementedType' object cannot be interpreted as an integer
    #
    #   + `raise NotImplementedError()`:
    #     This prevents triggering some fallback behavior. E.g., the built-in
    #     `list(X)` tries to call `len(X)` first, and executes a different code
    #     path if the method is not found or `NotImplemented` is returned, while
    #     raising an `NotImplementedError` will propagate and and make the call
    #     fail where it could have use `__iter__` to complete the call.
    #
    # Thus, the only two sensible things to do are
    #
    #   + **not** provide a default `__len__`.
    #
    #   + raise a `TypeError` instead, which is what Python uses when users call
    #     a method that is not defined on an object.
    #     (@ssnl verifies that this works on at least Python 3.7.)


class SequentialSampler(Sampler[int]):
    r"""Samples elements sequentially, always in the same order.

    Args:
        data_source (Dataset): dataset to sample from
    """
    data_source: Sized

    def __init__(self, data_source):
        self.data_source = data_source

    def __iter__(self):
        return iter(range(len(self.data_source)))

    def __len__(self) -> int:
        return len(self.data_source)


class RandomSampler(Sampler[int]):
    r"""Samples elements randomly. If without replacement, then sample from a shuffled dataset.
    If with replacement, then user can specify :attr:`num_samples` to draw.

    Args:
        data_source (Dataset): dataset to sample from
        replacement (bool): samples are drawn on-demand with replacement if ``True``, default=``False``
        num_samples (int): number of samples to draw, default=`len(dataset)`. This argument
            is supposed to be specified only when `replacement` is ``True``.
        generator (Generator): Generator used in sampling.
    """
    data_source: Sized
    replacement: bool

    def __init__(self, data_source: Sized, replacement: bool = False,
                 num_samples: Optional[int] = None, generator=None) -> None:
        self.data_source = data_source
        self.replacement = replacement
        self._num_samples = num_samples
        self.generator = generator

        if not isinstance(self.replacement, bool):
            raise TypeError("replacement should be a boolean value, but got "
                            "replacement={}".format(self.replacement))

        if self._num_samples is not None and not replacement:
            raise ValueError("With replacement=False, num_samples should not be specified, "
                             "since a random permute will be performed.")

        if not isinstance(self.num_samples, int) or self.num_samples <= 0:
            raise ValueError("num_samples should be a positive integer "
                             "value, but got num_samples={}".format(self.num_samples))

    @property
    def num_samples(self) -> int:
        # dataset size might change at runtime
        if self._num_samples is None:
            return len(self.data_source)
        return self._num_samples

    def __iter__(self):
        n = len(self.data_source)
        if self.generator is None:
            generator = torch.Generator()
            generator.manual_seed(int(torch.empty((), dtype=torch.int64).random_().item()))
        else:
            generator = self.generator
        if self.replacement:
            for _ in range(self.num_samples // 32):
                yield from torch.randint(high=n, size=(32,), dtype=torch.int64, generator=generator).tolist()
            yield from torch.randint(high=n, size=(self.num_samples % 32,), dtype=torch.int64, generator=generator).tolist()
        else:
            lst = torch.randperm(n, generator=self.generator).tolist()
            print('random sequence:', lst[:10], 'last one of sequence:', lst[-1])
            yield from lst

    def __len__(self):
        return self.num_samples


class SubsetRandomSampler(Sampler[int]):
    r"""Samples elements randomly from a given list of indices, without replacement.

    Args:
        indices (sequence): a sequence of indices
        generator (Generator): Generator used in sampling.
    """
    indices: Sequence[int]

    def __init__(self, indices: Sequence[int], generator=None) -> None:
        self.indices = indices
        self.generator = generator

    def __iter__(self):
        return (self.indices[i] for i in torch.randperm(len(self.indices), generator=self.generator))

    def __len__(self):
        return len(self.indices)


class WeightedRandomSampler(Sampler[int]):
    r"""Samples elements from ``[0,..,len(weights)-1]`` with given probabilities (weights).

    Args:
        weights (sequence)   : a sequence of weights, not necessary summing up to one
        num_samples (int): number of samples to draw
        replacement (bool): if ``True``, samples are drawn with replacement.
            If not, they are drawn without replacement, which means that when a
            sample index is drawn for a row, it cannot be drawn again for that row.
        generator (Generator): Generator used in sampling.

    Example:
        >>> list(WeightedRandomSampler([0.1, 0.9, 0.4, 0.7, 3.0, 0.6], 5, replacement=True))
        [4, 4, 1, 4, 5]
        >>> list(WeightedRandomSampler([0.9, 0.4, 0.05, 0.2, 0.3, 0.1], 5, replacement=False))
        [0, 1, 4, 3, 2]
    """
    weights: Tensor
    num_samples: int
    replacement: bool

    def __init__(self, weights: Sequence[float], num_samples: int,
                 replacement: bool = True, generator=None) -> None:
        if not isinstance(num_samples, _int_classes) or isinstance(num_samples, bool) or \
                num_samples <= 0:
            raise ValueError("num_samples should be a positive integer "
                             "value, but got num_samples={}".format(num_samples))
        if not isinstance(replacement, bool):
            raise ValueError("replacement should be a boolean value, but got "
                             "replacement={}".format(replacement))
        self.weights = torch.as_tensor(weights, dtype=torch.double)
        self.num_samples = num_samples
        self.replacement = replacement
        self.generator = generator

    def __iter__(self):
        rand_tensor = torch.multinomial(self.weights, self.num_samples, self.replacement, generator=self.generator)
        return iter(rand_tensor.tolist())

    def __len__(self):
        return self.num_samples


class BatchSampler(Sampler[List[int]]):
    r"""Wraps another sampler to yield a mini-batch of indices.

    Args:
        sampler (Sampler or Iterable): Base sampler. Can be any iterable object
        batch_size (int): Size of mini-batch.
        drop_last (bool): If ``True``, the sampler will drop the last batch if
            its size would be less than ``batch_size``

    Example:
        >>> list(BatchSampler(SequentialSampler(range(10)), batch_size=3, drop_last=False))
        [[0, 1, 2], [3, 4, 5], [6, 7, 8], [9]]
        >>> list(BatchSampler(SequentialSampler(range(10)), batch_size=3, drop_last=True))
        [[0, 1, 2], [3, 4, 5], [6, 7, 8]]
    """

    def __init__(self, sampler: Sampler[int], batch_size: int, drop_last: bool) -> None:
        # Since collections.abc.Iterable does not check for `__getitem__`, which
        # is one way for an object to be an iterable, we don't do an `isinstance`
        # check here.
        if not isinstance(batch_size, _int_classes) or isinstance(batch_size, bool) or \
                batch_size <= 0:
            raise ValueError("batch_size should be a positive integer value, "
                             "but got batch_size={}".format(batch_size))
        if not isinstance(drop_last, bool):
            raise ValueError("drop_last should be a boolean value, but got "
                             "drop_last={}".format(drop_last))
        self.sampler = sampler
        self.batch_size = batch_size
        self.drop_last = drop_last

    def __iter__(self):
        batch = []
        for idx in self.sampler:
            batch.append(idx)
            if len(batch) == self.batch_size:
                yield batch
                batch = []
        if len(batch) > 0 and not self.drop_last:
            yield batch

    def __len__(self):
        # Can only be called if self.sampler has __len__ implemented
        # We cannot enforce this condition, so we turn off typechecking for the
        # implementation below.
        # Somewhat related: see NOTE [ Lack of Default `__len__` in Python Abstract Base Classes ]
        if self.drop_last:
            return len(self.sampler) // self.batch_size  # type: ignore
        else:
            return (len(self.sampler) + self.batch_size - 1) // self.batch_size  # type: ignore

class ISASampler(Sampler[int]):
    r"""ISASampler using isa algorithm from \[ICLR16\] "ONLINE BATCH SELECTION FOR FASTER TRAINING OF NEURAL NETWORKS"

    Args:
        data_source (Dataset): dataset to sample from
        replacement (bool): samples are drawn on-demand with replacement if ``True``, default=``False``
        num_samples (int): number of samples to draw, default=`len(dataset)`. This argument
            is supposed to be specified only when `replacement` is ``True``.
        generator (Generator): Generator used in sampling.
    """
    data_source: Sized
    replacement: bool
    

    def __init__(self, data_source: Sized, lst=None, nepochs = 90, cur_epoch=0, replacement: bool = False,
                 num_samples: Optional[int] = None, generator=None) -> None:
        self.data_source = data_source
        self.replacement = replacement
        self._num_samples = num_samples
        self.generator = generator

        if not isinstance(self.replacement, bool):
            raise TypeError("replacement should be a boolean value, but got "
                            "replacement={}".format(self.replacement))

        if self._num_samples is not None and not replacement:
            raise ValueError("With replacement=False, num_samples should not be specified, "
                             "since a random permute will be performed.")

        if not isinstance(self.num_samples, int) or self.num_samples <= 0:
            raise ValueError("num_samples should be a positive integer "
                             "value, but got num_samples={}".format(self.num_samples))
        
        print('init isasampler')
        self.ivpersample = lst

        # online ISA related params
        self.isr = 0.8  # sampling ratio per epoch
        self.s0 = 1e2
        self.send = 1e2
        self.se_decline_factor = 1. if self.s0==self.send else math.exp(math.log(self.send/self.s0)/nepochs)
        self.cur_epoch = cur_epoch
        
        # introduced
        self.threshold = 0.7 # a threshold to distinguish important and unimportant samples
        

    @property
    def num_samples(self) -> int:
        # dataset size might change at runtime
        if self._num_samples is None:
            return len(self.data_source)
        return self._num_samples

    def __iter__(self):
        print('first 10 samples of ivpersample, cur_epoch:', self.ivpersample[:10], self.cur_epoch) # debug 
        yield from self.get_epoch_training_id_lst()

    def __len__(self):
        return self.num_samples
    
    def get_epoch_training_id_lst(self):
        # return a list of sample ID idx according to self.ivpersample
        sorted_ivpersample_idx = sorted(range(len(self.ivpersample)), key=lambda x: self.ivpersample[x], reverse=True)
        se = self.s0 * math.pow(self.se_decline_factor, self.cur_epoch)
        noimg = len(self.data_source)
        mult = math.exp(math.log(se) / noimg)
        p_lst = [1.] * noimg
        for i in range(1, noimg):
            p_lst[i] = p_lst[i-1] / mult
        # regulization for p_lst
        tmp_s = sum(p_lst)
        p_lst = [p/tmp_s for p in p_lst]
        # fill out accu_lst which denotes accumulation of p_lst
        accu_lst = [1.] * noimg
        accu_lst[0] = p_lst[0]
        for i in range(1, noimg):
            accu_lst[i] = accu_lst[i-1] + p_lst[i]
        
        selected_len = int(noimg * self.isr)
        important_len = int(selected_len * self.threshold)
        

        ret_lst = []
        if self.cur_epoch!=0:
            for i in range(selected_len):
                r = random.random()
                idx = self.find_minidx_from_accu(r, accu_lst)
                imgidx = sorted_ivpersample_idx[idx]
                if idx <= important_len:
                    ret_lst.append((imgidx, 1)) # 1 h-sample
                else:
                    ret_lst.append((imgidx, 0)) # 0 l-sample
        else:
            lst = list(range(noimg))
            random.shuffle(lst)
            for imgidx in lst:
                ret_lst.append((imgidx, 1))
        return ret_lst
    
    def find_minidx_from_accu(self, r, accu_lst):
        lft = 0
        rgt = len(accu_lst)
        mid = 0
        while(lft <= rgt):
            mid = lft + ((rgt-lft) >> 1)
            if accu_lst[mid] >= r:
                rgt = mid -1
            else:
                lft = mid +1
        return mid


class SBPSampler(Sampler[int]):
    r"""ISASampler using isa algorithm from \[ICLR16\] "ONLINE BATCH SELECTION FOR FASTER TRAINING OF NEURAL NETWORKS"

    Args:
        data_source (Dataset): dataset to sample from
        replacement (bool): samples are drawn on-demand with replacement if ``True``, default=``False``
        num_samples (int): number of samples to draw, default=`len(dataset)`. This argument
            is supposed to be specified only when `replacement` is ``True``.
        generator (Generator): Generator used in sampling.
    """
    data_source: Sized
    replacement: bool
    

    def __init__(self, data_source: Sized, lst=None, cur_epoch=0, replacement: bool = False,
                 num_samples: Optional[int] = None, generator=None,
                 reuse_factor=3, beta=1, his_len=1024, warm_up=5, batch_size=256) -> None:
        self.data_source = data_source
        self.replacement = replacement
        self._num_samples = num_samples
        self.generator = generator

        if not isinstance(self.replacement, bool):
            raise TypeError("replacement should be a boolean value, but got "
                            "replacement={}".format(self.replacement))

        if self._num_samples is not None and not replacement:
            raise ValueError("With replacement=False, num_samples should not be specified, "
                             "since a random permute will be performed.")

        if not isinstance(self.num_samples, int) or self.num_samples <= 0:
            raise ValueError("num_samples should be a positive integer "
                             "value, but got num_samples={}".format(self.num_samples))
        
        print('init sbpsampler')
        self.ivpersample = lst

        self.cur_epoch = cur_epoch
        self.reuse_factor = reuse_factor
        self.beta = beta
        self.his_len = his_len
        self.warm_up = warm_up
        self.batch_size = batch_size

        self.his_loss = collections.deque([], maxlen=self.his_len)

        # introduced
        self.threshold = 0.7 # a threshold to distinguish important and unimportant samples

        # final selected id list
        # self.ret_lst = self.get_epoch_training_id_lst()
        # self.record_idx = [i for i in range(0, len(self.data_source), 2500)]
        
        # self.f = open('record_idx_ivpersample_selected.txt', 'a+')
        # print('self.record_idx:', self.record_idx, file=self.f)
        

    @property
    def num_samples(self) -> int:
        # dataset size might change at runtime
        if self._num_samples is None:
            return len(self.data_source)
        return self._num_samples

    def __iter__(self):
        print('first 10 samples of ivpersample, cur_epoch:', self.ivpersample[:10], self.cur_epoch) # debug
        # final selected id list
        # print(f'* cur_epoch: {self.cur_epoch}, selected idx ivpersample:{[self.ivpersample[idx] for idx in self.record_idx]}', file=self.f)
        self.ret_lst = self.get_epoch_training_id_lst()
        # if len(self.ret_lst[0])==2:
        #     selected_idx = [item[0] for item in self.ret_lst]
        # else:
        #     selected_idx = self.ret_lst
        # print(f'* selected or not: {[idx in selected_idx for idx in self.record_idx]}', file=self.f)
        # print('***')
        # print('self.ret_lst:', self.ret_lst)
        yield from self.ret_lst

    def __len__(self):
        return self.num_samples
        # return len(self.ret_lst)
    
    def get_epoch_training_id_lst(self):
        st = time.time()
        ret_lst = []
        if (self.cur_epoch < self.warm_up) or (self.cur_epoch % self.reuse_factor == 0):
            ret_lst = list(self.generate_full_random_permutation())
        else:
            ret_lst = self.generate_selective_random_permutation()
        ret_lst = self.add_importance_flag(ret_lst, self.cur_epoch, self.warm_up, self.reuse_factor)
        print(f'epoch {self.cur_epoch} has selected {len(ret_lst)} sample losses.')
        print(f'generate selected id over, spent time: {time.time() - st}')
        return ret_lst


    def generate_full_random_permutation(self):
        n = len(self.data_source)
        if self.generator is None:
            generator = torch.Generator()
            generator.manual_seed(int(torch.empty((), dtype=torch.int64).random_().item()))
        else:
            generator = self.generator
        if self.replacement:
            for _ in range(self.num_samples // 32):
                yield from torch.randint(high=n, size=(32,), dtype=torch.int64, generator=generator).tolist()
            yield from torch.randint(high=n, size=(self.num_samples % 32,), dtype=torch.int64, generator=generator).tolist()
        else:
            lst = torch.randperm(n, generator=self.generator).tolist()
            print('FULL random sequence[:10]:', lst[:10], 'last one of sequence:', lst[-1])
            yield from lst
    
    def generate_selective_random_permutation(self):
        ret_lst = []
        tmp_ret_lst = list(self.generate_full_random_permutation())
        len_tmp_ret_lst = len(tmp_ret_lst)
        for i in range(0, len_tmp_ret_lst, self.batch_size):
            batch_lst = tmp_ret_lst[i:i+self.batch_size]
            batch_loss = [self.ivpersample[idx] for idx in batch_lst]
            # self.his_loss.extend(batch_loss) # append new loss
            if len(self.his_loss) < self.his_len:
                ret_lst.extend(batch_lst)
            else:
                selection_probabilities = self._get_selection_probabilities(batch_loss, self.beta)
                selection = selection_probabilities > np.random.random(*selection_probabilities.shape)
                ret_lst.extend([batch_lst[idx] for idx in list(np.argwhere(selection).flatten())])
            self.his_loss.extend(batch_loss) # append new loss
        return ret_lst
    
    def _get_selection_probabilities(self, loss, beta):
        percentiles = self._percentiles(self.his_loss, loss)
        # return percentiles ** 2
        return np.power(percentiles, beta)
    
    def _percentiles(self, hist_values, values_to_search):
        # TODO Speed up this again. There is still a visible overhead in training. 
        hist_values, values_to_search = np.asarray(hist_values), np.asarray(values_to_search)

        percentiles_values = np.percentile(hist_values, range(100))
        sorted_loss_idx = sorted(range(len(values_to_search)), key=lambda k: values_to_search[k])
        counter = 0
        percentiles_by_loss = [0] * len(values_to_search)
        for idx, percentiles_value in enumerate(percentiles_values):
            while values_to_search[sorted_loss_idx[counter]] < percentiles_value:
                percentiles_by_loss[sorted_loss_idx[counter]] = idx
                counter += 1
                if counter == len(values_to_search) : break
            if counter == len(values_to_search) : break
        return np.array(percentiles_by_loss)/100
    
    def add_importance_flag(self, ori_ret_lst, cur_epoch, warm_up, reuse_factor):
        # sort idx of ori_ret_lst decreasing by loss value
        total_idx = len(ori_ret_lst)
        sorted_idx = sorted(range(total_idx), key=lambda idx:self.ivpersample[idx], reverse=True)
        important_lst_len = int(total_idx * self.threshold)

        if cur_epoch >= warm_up and cur_epoch%reuse_factor:
            for idx in sorted_idx[:important_lst_len]:
                ori_ret_lst[idx] = (ori_ret_lst[idx], 1)
            for idx in sorted_idx[important_lst_len:]:
                ori_ret_lst[idx] = (ori_ret_lst[idx], 0)
        else:
            for idx in sorted_idx[:important_lst_len]:
                ori_ret_lst[idx] = (ori_ret_lst[idx], 1)
            for idx in sorted_idx[important_lst_len:]:
                ori_ret_lst[idx] = (ori_ret_lst[idx], 1)
        return ori_ret_lst


class DistributedSBPSampler(Sampler[int]):
    r"""ISASampler using isa algorithm from \[ICLR16\] "ONLINE BATCH SELECTION FOR FASTER TRAINING OF NEURAL NETWORKS"

    Args:
        data_source (Dataset): dataset to sample from
        replacement (bool): samples are drawn on-demand with replacement if ``True``, default=``False``
        num_samples (int): number of samples to draw, default=`len(dataset)`. This argument
            is supposed to be specified only when `replacement` is ``True``.
        generator (Generator): Generator used in sampling.
    """
    data_source: Sized
    replacement: bool
    

    def __init__(self, data_source: Sized, lst=None, cur_epoch=0, replacement: bool = False,
                 num_samples: Optional[int] = None, generator=None,
                 reuse_factor=3, beta=1, his_len=1024, warm_up=5, batch_size=256) -> None:
        self.data_source = data_source
        self.replacement = replacement
        self._num_samples = num_samples
        self.generator = generator

        if not isinstance(self.replacement, bool):
            raise TypeError("replacement should be a boolean value, but got "
                            "replacement={}".format(self.replacement))

        if self._num_samples is not None and not replacement:
            raise ValueError("With replacement=False, num_samples should not be specified, "
                             "since a random permute will be performed.")

        if not isinstance(self.num_samples, int) or self.num_samples <= 0:
            raise ValueError("num_samples should be a positive integer "
                             "value, but got num_samples={}".format(self.num_samples))
        
        print('init distributedsbpsampler')
        self.ivpersample = lst

        self.cur_epoch = cur_epoch
        self.reuse_factor = reuse_factor
        self.beta = beta
        self.his_len = his_len
        self.warm_up = warm_up
        self.batch_size = batch_size

        self.his_loss = collections.deque([], maxlen=self.his_len)

        # introduced
        self.threshold = 0.7 # a threshold to distinguish important and unimportant samples

        # final selected id list
        # self.ret_lst = self.get_epoch_training_id_lst()
        self.num_replicas = dist.get_world_size()
        self.rank = dist.get_rank()
        if len(self.data_source) % self.num_replicas:
            self._num_samples = math.ceil((len(self.data_source) - self.num_replicas) / self.num_replicas)
        else:
            self._num_samples = math.ceil(len(self.data_source) / self.num_replicas)
        self.total_size = self._num_samples * self.num_replicas
        

    @property
    def num_samples(self) -> int:
        # dataset size might change at runtime
        if self._num_samples is None:
            return len(self.data_source)
        return self._num_samples

    def __iter__(self):
        print('first 10 samples of ivpersample, cur_epoch:', self.ivpersample[:10], self.cur_epoch) # debug
        # final selected id list
        self.ret_lst = self.get_epoch_training_id_lst()
        # print('***')
        # print('self.ret_lst:', self.ret_lst)
        yield from self.ret_lst

    def __len__(self):
        return self.num_samples
        # return len(self.ret_lst)
    
    def get_epoch_training_id_lst(self):
        st = time.time()
        ret_lst = []
        if (self.cur_epoch < self.warm_up) or (self.cur_epoch % self.reuse_factor == 0):
            ret_lst = list(self.generate_full_random_permutation())
        else:
            ret_lst = self.generate_selective_random_permutation()
            print(f'0. len of ret_lst: {len(ret_lst)}')
        ret_lst = self.add_importance_flag(ret_lst, self.cur_epoch, self.warm_up, self.reuse_factor)
        print(f'0. len of ret_lst: {len(ret_lst)}')
        ret_lst = ret_lst[:self.total_size]
        print(f'0. len of ret_lst: {len(ret_lst)}')
        ret_lst = ret_lst[self.rank:self.total_size:self.num_replicas]
        print(f'0. len of ret_lst: {len(ret_lst)}')
        print(f'Rank {self.rank}, epoch {self.cur_epoch} has selected {len(ret_lst)} sample losses.')
        print(f'Rank {self.rank}, generate selected id over, spent time: {time.time() - st}')
        return ret_lst


    def generate_full_random_permutation(self):
        n = len(self.data_source)
        if self.generator is None:
            generator = torch.Generator()
            generator.manual_seed(self.cur_epoch)
            print(f'Rank{self.rank} set distributedSBPsampler generator seed: {self.cur_epoch}')
            # generator.manual_seed(int(torch.empty((), dtype=torch.int64).random_().item()))
        else:
            generator = self.generator
        if self.replacement:
            for _ in range(self.num_samples // 32):
                yield from torch.randint(high=n, size=(32,), dtype=torch.int64, generator=generator).tolist()
            yield from torch.randint(high=n, size=(self.num_samples % 32,), dtype=torch.int64, generator=generator).tolist()
        else:
            lst = torch.randperm(n, generator=generator).tolist()
            print('FULL random sequence[:10]:', lst[:10], 'last one of sequence:', lst[-1])
            yield from lst
    
    def generate_selective_random_permutation(self):
        print(f'calling generate_selective_random_permutation()')
        ret_lst = []
        tmp_ret_lst = list(self.generate_full_random_permutation())
        len_tmp_ret_lst = len(tmp_ret_lst)
        # dist-related
        np.random.seed(self.cur_epoch)
        for i in range(0, len_tmp_ret_lst, self.batch_size):
            batch_lst = tmp_ret_lst[i:i+self.batch_size]
            batch_loss = [self.ivpersample[idx] for idx in batch_lst]
            # self.his_loss.extend(batch_loss) # append new loss
            if len(self.his_loss) < self.his_len:
                ret_lst.extend(batch_lst)
            else:
                selection_probabilities = self._get_selection_probabilities(batch_loss, self.beta)
                selection = selection_probabilities > np.random.random(*selection_probabilities.shape)
                ret_lst.extend([batch_lst[idx] for idx in list(np.argwhere(selection).flatten())])
            self.his_loss.extend(batch_loss) # append new loss
        return ret_lst
    
    def _get_selection_probabilities(self, loss, beta):
        percentiles = self._percentiles(self.his_loss, loss)
        # return percentiles ** 2
        return np.power(percentiles, beta)
    
    def _percentiles(self, hist_values, values_to_search):
        # TODO Speed up this again. There is still a visible overhead in training. 
        hist_values, values_to_search = np.asarray(hist_values), np.asarray(values_to_search)

        percentiles_values = np.percentile(hist_values, range(100))
        sorted_loss_idx = sorted(range(len(values_to_search)), key=lambda k: values_to_search[k])
        counter = 0
        percentiles_by_loss = [0] * len(values_to_search)
        for idx, percentiles_value in enumerate(percentiles_values):
            while values_to_search[sorted_loss_idx[counter]] < percentiles_value:
                percentiles_by_loss[sorted_loss_idx[counter]] = idx
                counter += 1
                if counter == len(values_to_search) : break
            if counter == len(values_to_search) : break
        return np.array(percentiles_by_loss)/100
    
    def add_importance_flag(self, ori_ret_lst, cur_epoch, warm_up, reuse_factor):
        # sort idx of ori_ret_lst decreasing by loss value
        total_idx = len(ori_ret_lst)
        sorted_idx = sorted(range(total_idx), key=lambda idx:self.ivpersample[idx], reverse=True)
        important_lst_len = int(total_idx * self.threshold)

        if cur_epoch >= warm_up and cur_epoch%reuse_factor:
            for idx in sorted_idx[:important_lst_len]:
                ori_ret_lst[idx] = (ori_ret_lst[idx], 1)
            for idx in sorted_idx[important_lst_len:]:
                ori_ret_lst[idx] = (ori_ret_lst[idx], 0)
        else:
            for idx in sorted_idx[:important_lst_len]:
                ori_ret_lst[idx] = (ori_ret_lst[idx], 1)
            for idx in sorted_idx[important_lst_len:]:
                ori_ret_lst[idx] = (ori_ret_lst[idx], 1)
        return ori_ret_lst


