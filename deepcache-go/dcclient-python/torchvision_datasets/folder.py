import time
import json
import multiprocessing
from .vision import VisionDataset

from PIL import Image

import os
import sys
import os.path
from typing import Any, Callable, cast, Dict, List, Optional, Tuple

import six
import multiprocessing as mp

import grpc
from . import deepcache_pb2
from . import deepcache_pb2_grpc
# Set GRPC communication IP and port
GPRC_ADDR_PORT = "127.0.0.1:18180"


def has_file_allowed_extension(filename: str, extensions: Tuple[str, ...]):
    """
    Checks if a file is an allowed extension.

    Args:
        filename (string): path to a file
        extensions (tuple of strings): extensions to consider (lowercase)

    Returns:
        bool: True if the filename ends with one of given extensions
    """
    return filename.lower().endswith(extensions)


def is_image_file(filename: str):
    """
    Checks if a file is an allowed image extension.

    Args:
        filename (string): path to a file

    Returns:
        bool: True if the filename ends with a known image extension
    """
    return has_file_allowed_extension(filename, IMG_EXTENSIONS)


def make_dataset(
    directory: str,
    class_to_idx: Dict[str, int],
    extensions: Optional[Tuple[str, ...]] = None,
    is_valid_file: Optional[Callable[[str], bool]] = None,
):
    """Generates a list of samples of a form (path_to_sample, class).

    Args:
        directory (str): root dataset directory
        class_to_idx (Dict[str, int]): dictionary mapping class name to class index
        extensions (optional): A list of allowed extensions.
            Either extensions or is_valid_file should be passed. Defaults to None.
        is_valid_file (optional): A function that takes path of a file
            and checks if the file is a valid file
            (used to check of corrupt files) both extensions and
            is_valid_file should not be passed. Defaults to None.

    Raises:
        ValueError: In case ``extensions`` and ``is_valid_file`` are None or both are not None.

    Returns:
        List[Tuple[str, int]]: samples of a form (path_to_sample, class)
    """
    instances = []
    directory = os.path.expanduser(directory)
    both_none = extensions is None and is_valid_file is None
    both_something = extensions is not None and is_valid_file is not None
    if both_none or both_something:
        raise ValueError(
            "Both extensions and is_valid_file cannot be None or not None at the same time")
    if extensions is not None:
        def is_valid_file(x: str) -> bool:
            return has_file_allowed_extension(x, cast(Tuple[str, ...], extensions))
    is_valid_file = cast(Callable[[str], bool], is_valid_file)
    for target_class in sorted(class_to_idx.keys()):
        class_index = class_to_idx[target_class]
        target_dir = os.path.join(directory, target_class)
        if not os.path.isdir(target_dir):
            continue
        for root, _, fnames in sorted(os.walk(target_dir, followlinks=True)):
            for fname in sorted(fnames):
                path = os.path.join(root, fname)
                if is_valid_file(path):
                    item = path, class_index
                    instances.append(item)
    return instances


class TrainDatasetFolder(VisionDataset):
    def __init__(
            self,
            root: str,
            loader: Callable[[str], Any],
            extensions: Optional[Tuple[str, ...]] = None,
            transform: Optional[Callable] = None,
            target_transform: Optional[Callable] = None,
            is_valid_file: Optional[Callable[[str], bool]] = None,
            imgidx2pil=None,
            open_pil_cache=False,
            grpc_port="127.0.0.1:18180",
    ):
        super(TrainDatasetFolder, self).__init__(root, transform=transform,
                                                 target_transform=target_transform)
        classes, class_to_idx = self._find_classes(self.root)
        samples = self.make_dataset(
            self.root, class_to_idx, extensions, is_valid_file)
        if len(samples) == 0:
            msg = "Found 0 files in subfolders of: {}\n".format(self.root)
            if extensions is not None:
                msg += "Supported extensions are: {}".format(
                    ",".join(extensions))
            raise RuntimeError(msg)

        self.loader = loader
        self.extensions = extensions

        self.classes = classes
        self.class_to_idx = class_to_idx
        self.samples = samples
        self.targets = [s[1] for s in samples]

        # add a map for fast build PIL image from string
        self.imgidx2pil = imgidx2pil
        self.open_pil_cache = open_pil_cache
        self.pilcached_size_upbound = int(len(samples) * 0.2)
        # print('self.pilcached_size_upbound =', self.pilcached_size_upbound)

        # self.channel = grpc.insecure_channel(grpc_port,options=[
        #                            ('grpc.enable_retries', 1),
        #                            ('grpc.keepalive_timeout_ms', 100000),
        #                            ('grpc.max_receive_message_length', 20 * 1024 * 1024), # max grpc size 20MB
        #                        ])
        # self._stub = deepcache_pb2_grpc.OperatorStub(self.channel)

        # # st = time.time()
        # response = self._stub.DCSubmit(deepcache_pb2.DCRequest(type=deepcache_pb2.get_cache_info), timeout=1000)
        # self.cache_size = response.cache_size
        # self.nsamples = response.nsamples

        # print('* server info (grpc): cache_size, total samples =', self.cache_size, self.nsamples, response.msg)
        # self.channel.close()
        print('TrainDatasetFolder initialized')

    @staticmethod
    def make_dataset(
        directory: str,
        class_to_idx: Dict[str, int],
        extensions: Optional[Tuple[str, ...]] = None,
        is_valid_file: Optional[Callable[[str], bool]] = None,
    ):
        return make_dataset(directory, class_to_idx, extensions=extensions, is_valid_file=is_valid_file)

    def _find_classes(self, dir: str):
        """
        Finds the class folders in a dataset.

        Args:
            dir (string): Root directory path.

        Returns:
            tuple: (classes, class_to_idx) where classes are relative to (dir), and class_to_idx is a dictionary.

        Ensures:
            No class is a subdirectory of another.
        """
        classes = [d.name for d in os.scandir(dir) if d.is_dir()]
        classes.sort()
        class_to_idx = {cls_name: i for i, cls_name in enumerate(classes)}
        return classes, class_to_idx

    def __getitem__(self, index: int, stub=None, fakecache=False):
        """
        Args:
            index (int): Index

        Returns:
            tuple: (sample, target) where target is class_index of the target class.
        """

        # st = time.time()
        sample, target, imgidx, ishit = self.rpyc_loader(index, stub, fakecache)
        # print(f'rpc_loader {index}: {time.time() - st}')

        # quiver or isa
        if imgidx == -1:
            return sample, target, imgidx, ishit

        if self.transform is not None:
            # st = time.time()
            sample = self.transform(sample)
            # print(f'transform {index}: {time.time() - st}\n')
        if self.target_transform is not None:
            target = self.target_transform(target)
        return sample, target, imgidx, ishit

    def __len__(self) -> int:
        return len(self.samples)

    # @profile
    def rpyc_loader(self, _imgidx, stub, fakecache=False):
        # st = time.time()
        response = stub.DCSubmit(deepcache_pb2.DCRequest(
            type=deepcache_pb2.readimg_byidx, imgidx=_imgidx, id=int(fakecache)), timeout=1000)
            # id=0 not fakecache，=1 fakecache，default value is0；
        # print(f'grpc read_imgidx {_imgidx}: {time.time() - st}')

        # st = time.time()
        self.cache_size = response.cache_size
        img = response.data
        target = response.clsidx
        imgidx = response.imgidx
        ishit = response.is_hit

        # quiver
        if imgidx == -1:
            return img, target, imgidx, ishit

        if self.open_pil_cache:
            # st = time.time()
            if response.is_hit:
                if imgidx in self.imgidx2pil:
                    img = self.imgidx2pil[imgidx]
                    # print(f'{imgidx} server cache hit and read pil from bytes2pil.')
                else:
                    img = self.construct_PIL_from_bytes(img)
                    if len(self.imgidx2pil) < self.pilcached_size_upbound:
                        self.imgidx2pil[imgidx] = img
                    # print(f'{imgidx} server cache hit and just put into cache bytes2pil.')
            else:
                img = self.construct_PIL_from_bytes(img)
                if len(self.imgidx2pil) < self.pilcached_size_upbound:
                    self.imgidx2pil[imgidx] = img
                # print(f'{imgidx} server cache NOT hit and just put into cache bytes2pil.')
        else:
            # st = time.time()
            if len(img) == 0:
                print(imgidx, target, img, ishit)
            img = self.construct_PIL_from_bytes(img)
            # print(f'construct pil from byte: {time.time() - st}')
        return img, target, imgidx, ishit

    def construct_PIL_from_bytes(self, bytes_):
        if len(bytes_) == 0:
            raise Exception('bytes_ is none')
        with six.BytesIO(bytes_) as stream:
            img = Image.open(stream)
            img = img.convert('RGB')
        return img

    def rpyc_loader_t(self, imgidx_t, stub, full_access, fakecache=False):
        tmp_kv = {imgidx_t[0]: imgidx_t[1]}
        response = stub.DCSubmit(deepcache_pb2.DCRequest(
            type=deepcache_pb2.readimg_byidx_t, imgidx_t=json.dumps(tmp_kv), full_access=full_access, id=int(fakecache)), timeout=1000)
        # print(f'grpc read_imgidx {imgidx_t}')

        self.cache_size = response.cache_size
        img = response.data
        target = response.clsidx
        imgidx = response.imgidx
        ishit = response.is_hit
        # print(f'grpc abstract info after read_imgidx {imgidx}')
        # quiver
        if imgidx == -1:
            return img, target, imgidx, ishit

        img = self.construct_PIL_from_bytes(img)
        return img, target, imgidx, ishit

    def getitem(self, index_t: tuple, stub: None, full_access=False, fakecache=False):
        """
        Args:
            index_t (tuple): Index, Int(0 or 1)

        Returns:
            tuple: (sample, target) where target is class_index of the target class.
        """

        try:
            # st = time.time()
            sample, target, imgidx, ishit = self.rpyc_loader_t(
                index_t, stub, full_access, fakecache)
            # print(f'rpc_loader {index}: {time.time() - st}')

            # quiver or isa
            if imgidx == -1:
                return sample, target, imgidx, ishit

            if self.transform is not None:
                # st = time.time()
                sample = self.transform(sample)
                # print(f'transform {index}: {time.time() - st}\n')
            if self.target_transform is not None:
                target = self.target_transform(target)
            return sample, target, imgidx, ishit

        except BaseException as e:
            print(e)

    # def close(self):
    #     self.channel.close()


class DatasetFolder(VisionDataset):
    """A generic data loader where the samples are arranged in this way: ::
        root/class_x/xxx.ext
        root/class_x/xxy.ext
        root/class_x/xxz.ext
        root/class_y/123.ext
        root/class_y/nsdf3.ext
        root/class_y/asd932_.ext
    Args:
        root (string): Root directory path.
        loader (callable): A function to load a sample given its path.
        extensions (tuple[string]): A list of allowed extensions.
            both extensions and is_valid_file should not be passed.
        transform (callable, optional): A function/transform that takes in
            a sample and returns a transformed version.
            E.g, ``transforms.RandomCrop`` for images.
        target_transform (callable, optional): A function/transform that takes
            in the target and transforms it.
        is_valid_file (callable, optional): A function that takes path of a file
            and check if the file is a valid file (used to check of corrupt files)
            both extensions and is_valid_file should not be passed.
     Attributes:
        classes (list): List of the class names.
        class_to_idx (dict): Dict with items (class_name, class_index).
        samples (list): List of (sample path, class_index) tuples
        targets (list): The class_index value for each image in the dataset
    """

    def __init__(self, root, loader, extensions=None, transform=None,
                 target_transform=None, is_valid_file=None):
        super(DatasetFolder, self).__init__(root, transform=transform,
                                            target_transform=target_transform)
        classes, class_to_idx = self._find_classes(self.root)
        samples = make_dataset(self.root, class_to_idx,
                               extensions, is_valid_file)
        if len(samples) == 0:
            raise (RuntimeError("Found 0 files in subfolders of: " + self.root + "\n"
                                "Supported extensions are: " + ",".join(extensions)))

        self.loader = loader
        self.extensions = extensions

        self.classes = classes
        self.class_to_idx = class_to_idx
        self.samples = samples
        self.targets = [s[1] for s in samples]

    def _find_classes(self, dir):
        """
        Finds the class folders in a dataset.
        Args:
            dir (string): Root directory path.
        Returns:
            tuple: (classes, class_to_idx) where classes are relative to (dir), and class_to_idx is a dictionary.
        Ensures:
            No class is a subdirectory of another.
        """
        if sys.version_info >= (3, 5):
            # Faster and available in Python 3.5 and above
            classes = [d.name for d in os.scandir(dir) if d.is_dir()]
        else:
            classes = [d for d in os.listdir(
                dir) if os.path.isdir(os.path.join(dir, d))]
        classes.sort()
        class_to_idx = {classes[i]: i for i in range(len(classes))}
        return classes, class_to_idx

    def __getitem__(self, index, stub=None, fakecache=False):
        """
        Args:
            index (int): Index
        Returns:
            tuple: (sample, target) where target is class_index of the target class.
        """
        path, target = self.samples[index]
        sample = self.loader(path)
        if self.transform is not None:
            sample = self.transform(sample)
        if self.target_transform is not None:
            target = self.target_transform(target)

        return sample, target

    def __len__(self):
        return len(self.samples)


IMG_EXTENSIONS = ('.jpg', '.jpeg', '.png', '.ppm', '.bmp',
                  '.pgm', '.tif', '.tiff', '.webp')


def pil_loader(path: str):

    # open path as file to avoid ResourceWarning (https://github.com/python-pillow/Pillow/issues/835)
    with open(path, 'rb') as f:
        # st = time.time()
        img = Image.open(f)
        img = img.convert('RGB')
        # print(f'img.convert {path}: {time.time() - st}')
        return img

def accimage_loader(path: str):
    # import accimage
    # try:
    #     return accimage.Image(path)
    # except IOError:
    #     # Potentially a decoding problem, fall back to PIL.Image
    #     return pil_loader(path)
    return pil_loader(path)


def default_loader(path: str):
    from torchvision import get_image_backend
    if get_image_backend() == 'accimage':
        return accimage_loader(path)
    else:
        return pil_loader(path)


class ImageFolder(DatasetFolder):
    """A generic data loader where the images are arranged in this way: ::

        root/dog/xxx.png
        root/dog/xxy.png
        root/dog/[...]/xxz.png

        root/cat/123.png
        root/cat/nsdf3.png
        root/cat/[...]/asd932_.png

    Args:
        root (string): Root directory path.
        transform (callable, optional): A function/transform that  takes in an PIL image
            and returns a transformed version. E.g, ``transforms.RandomCrop``
        target_transform (callable, optional): A function/transform that takes in the
            target and transforms it.
        loader (callable, optional): A function to load an image given its path.
        is_valid_file (callable, optional): A function that takes path of an Image file
            and check if the file is a valid file (used to check of corrupt files)

     Attributes:
        classes (list): List of the class names sorted alphabetically.
        class_to_idx (dict): Dict with items (class_name, class_index).
        imgs (list): List of (image path, class_index) tuples
    """

    def __init__(
            self,
            root: str,
            transform: Optional[Callable] = None,
            target_transform: Optional[Callable] = None,
            loader: Callable[[str], Any] = default_loader,
            is_valid_file: Optional[Callable[[str], bool]] = None,
    ):
        super(ImageFolder, self).__init__(root, loader, IMG_EXTENSIONS if is_valid_file is None else None,
                                          transform=transform,
                                          target_transform=target_transform,
                                          is_valid_file=is_valid_file)
        self.imgs = self.samples


class TrainImageFolder(TrainDatasetFolder):
    """A generic data loader where the images are arranged in this way: ::

        root/dog/xxx.png
        root/dog/xxy.png
        root/dog/[...]/xxz.png

        root/cat/123.png
        root/cat/nsdf3.png
        root/cat/[...]/asd932_.png

    Args:
        root (string): Root directory path.
        transform (callable, optional): A function/transform that  takes in an PIL image
            and returns a transformed version. E.g, ``transforms.RandomCrop``
        target_transform (callable, optional): A function/transform that takes in the
            target and transforms it.
        loader (callable, optional): A function to load an image given its path.
        is_valid_file (callable, optional): A function that takes path of an Image file
            and check if the file is a valid file (used to check of corrupt files)

     Attributes:
        classes (list): List of the class names sorted alphabetically.
        class_to_idx (dict): Dict with items (class_name, class_index).
        imgs (list): List of (image path, class_index) tuples
    """

    def __init__(
            self,
            root: str,
            transform: Optional[Callable] = None,
            target_transform: Optional[Callable] = None,
            loader: Callable[[str], Any] = default_loader,
            is_valid_file: Optional[Callable[[str], bool]] = None,
            grpc_port: str = "127.0.0.1:18180",
    ):
        # add a map for fast build PIL image from string
        self.imgidx2pil = multiprocessing.Manager().dict()
        super(TrainImageFolder, self).__init__(root, loader, IMG_EXTENSIONS if is_valid_file is None else None,
                                               transform=transform,
                                               target_transform=target_transform,
                                               is_valid_file=is_valid_file,
                                               imgidx2pil=self.imgidx2pil,
                                               grpc_port=grpc_port)
        self.imgs = self.samples
