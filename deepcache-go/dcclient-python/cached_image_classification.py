import argparse
import os
import sys
import random
import shutil
import time
import warnings

import torch
import torch.nn as nn
import torch.nn.parallel
import torch.backends.cudnn as cudnn
import torch.distributed as dist
import torch.optim
import torch.multiprocessing as mp
# import torch.utils.data
# import torch.utils.data.distributed
import torchvision.transforms as transforms
# import torchvision.datasets as datasets
import torchvision.models as models
from models_cifar10 import *
import requests

# Custom torch interface
import torch_cli.utils_data as dc_data
import torch_cli.datasets as dc_datasets

# from apex import amp # accelerate computing
# from apex.parallel import DistributedDataParallel

import os
# os.environ["CUDA_VISIBLE_DEVICES"] = "2,3"
from sbp import SelectiveBackPropagation

# REQ_HTTP_ADDR = "http://127.0.0.1:18182/"
# REQ_HTTP_ADDR_CACHE = REQ_HTTP_ADDR+"cache"
# REQ_HTTP_ADDR_STATISTIC = REQ_HTTP_ADDR+"statistic"

model_names = sorted(name for name in models.__dict__
                     if name.islower() and not name.startswith("__")
                     and callable(models.__dict__[name]))
last_remote_peer_hit = 0
start_count_iter = 5

def parse_args():
    parser = argparse.ArgumentParser(description='PyTorch ImageNet Training')
    parser.add_argument('--req-addr', default="http://127.0.0.1:18182/",
                        type=str, help='port when client rpc server in folder.py')

    parser.add_argument('--data', metavar='DIR',
                        default=None, help='path to dataset')
    parser.add_argument('-a', '--arch', metavar='ARCH', default='resnet18',
                        help='model architecture: ' +
                        ' | '.join(model_names) +
                        ' (default: resnet18)')
    parser.add_argument('-j', '--workers', default=4, type=int, metavar='N',
                        help='number of data loading workers (default: 4)')
    parser.add_argument('--epochs', default=90, type=int, metavar='N',
                        help='number of total epochs to run')
    parser.add_argument('--start-epoch', default=0, type=int, metavar='N',
                        help='manual epoch number (useful on restarts)')
    parser.add_argument('-b', '--batch-size', default=256, type=int,
                        metavar='N',
                        help='mini-batch size (default: 256), this is the total '
                        'batch size of all GPUs on the current node when '
                        'using Data Parallel or Distributed Data Parallel')
    parser.add_argument('--lr', '--learning-rate', default=0.1, type=float,
                        metavar='LR', help='initial learning rate', dest='lr')
    parser.add_argument('--momentum', default=0.9, type=float, metavar='M',
                        help='momentum')
    parser.add_argument('--wd', '--weight-decay', default=1e-4, type=float,
                        metavar='W', help='weight decay (default: 1e-4)',
                        dest='weight_decay')
    parser.add_argument('-p', '--print-freq', default=15, type=int,
                        metavar='N', help='print frequency (default: 10)')
    parser.add_argument('--resume', default='', type=str, metavar='PATH',
                        help='path to latest checkpoint (default: none)')
    parser.add_argument('-e', '--evaluate', dest='evaluate', action='store_true',
                        help='evaluate model on validation set')
    parser.add_argument('--pretrained', dest='pretrained', action='store_true',
                        help='use pre-trained model')
    parser.add_argument('--world-size', default=-1, type=int,
                        help='number of nodes for distributed training')
    parser.add_argument('--rank', default=-1, type=int,
                        help='node rank for distributed training')
    parser.add_argument('--dist-url', default='tcp://127.0.0.1:23456', type=str,
                        help='url used to set up distributed training')
    parser.add_argument('--dist-backend', default='nccl', type=str,
                        help='distributed backend')
    parser.add_argument('--seed', default=None, type=int,
                        help='seed for initializing training. ')
    parser.add_argument('--gpu', default=None, type=int,
                        help='GPU id to use.')
    parser.add_argument('--multiprocessing-distributed', action='store_true',
                        help='Use multi-processing distributed training to launch '
                        'N processes per node, which has N GPUs. This is the '
                        'fastest way to use PyTorch for either single node or '
                        'multi node data parallel training')
    parser.add_argument('--num-classes', default=1000, type=int,
                        help='default is 200 for tiny-imagenet-200 dataset')
    # for experimental, the real training process does not need this parameter
    parser.add_argument('-c', '--cache-type', metavar='CACHE_TYPE', default=None,
                        help="choose from lru | isa | coordl | quiver")
    parser.add_argument('-cr', '--cache-ratio', default=None, type=float,
                        help="cache ratio is necessary when cache_type is quiver")
    parser.add_argument('-ngpus', '--ngpus-per-node',
                        default=1, type=int, help='number of gpu to use')
    parser.add_argument('--apex', action='store_true',
                        help='use apex')
    # selective backpropagation
    parser.add_argument('--sbp', action='store_true', help='if using sbp')
    parser.add_argument('--beta', default=0.33, type=float,
                        help='beta value in SBP')
    parser.add_argument('--his-len', default=1024,
                        type=int, help='R value in SBP')
    parser.add_argument('--reuse-factor', default=1,
                        type=int, help='reuse-factor is n in paper')
    parser.add_argument('--warm-up', default=5, type=int, help='warm up')

    # auto run
    parser.add_argument('--grpc-port', default="127.0.0.1:18180",
                        type=str, help='port when client rpc server in folder.py')
    # fake cache read data
    parser.add_argument('--fakecache', action='store_true', help='go cache generates fake data')
    return parser.parse_args()


args = parse_args()

# connect the service side, get the training path
response = requests.get(args.req_addr+"statistic")
default_data_path = response.json()["dcserver_data_path"]
default_cache_type = response.json()['cache_type']
default_cache_ratio = response.json()['cache_ratio']
print(response.json()['cache_type'], response.json()['cache_ratio'])
# according to the server parameters, modify some default parameters
args.data = default_data_path
args.cache_type = default_cache_type
args.cache_ratio = default_cache_ratio
print(args.cache_type, args.cache_ratio)  # the same as before
print("Data path:" + args.data)

assert args.cache_type != None, 'for convinience of experiments, cache_type is necessary'
if args.cache_type == 'quiver':
    assert args.cache_ratio != 0, 'when cache_type is quiver, cache_ratio(-cr) is necessary'

best_acc1 = 0


def main():
    """ this main function is used to spawn training processes on each GPU """

    # reproduce the same results
    if args.seed is not None:
        random.seed(args.seed)
        torch.manual_seed(args.seed)
        cudnn.deterministic = True
    cudnn.benchmark = False

    # multiprocessing_distributed means multi-gpu on one single node
    # distributed means single node multiple gpus or multiple nodes
    args.distributed = args.world_size > 1 or args.multiprocessing_distributed

    # ngpus_per_node = torch.cuda.device_count()
    ngpus_per_node = args.ngpus_per_node
    print(f"ngpus_per_node:", ngpus_per_node)
    if args.multiprocessing_distributed:
        # args.world_size in command means # of node, in code means sum # of model replicas of all nodes
        args.world_size = ngpus_per_node * args.world_size
        # Use torch.multiprocessing.spawn to launch distributed processes on current node: the main_worker process
        mp.spawn(main_worker, nprocs=ngpus_per_node,
                 args=(ngpus_per_node, args))
    else:
        # one GPU on each node,
        # or user specify a gpu to train the network, thus not type --multiprocessing_distributed in command
        main_worker(args.gpu, ngpus_per_node, args)


def main_worker(gpu, ngpus_per_node, args):
    """ this main_worker function is used to create model, put model to GPUs, data loading and train by epochs"""
    response = requests.get(args.req_addr+"cache")
    info = response.json()
    imgidx2imgpath = info['imgidx2imgpath']

    global best_acc1
    args.gpu = gpu

    # when multiprocessing training or user specified used one gpu id, args.gpu will be gpu id.
    # when one gpu each node and user not specify gpu, args.gpu will be None.
    if args.gpu is not None:
        print("Use GPU: {} for training".format(args.gpu))

    if args.distributed:
        if args.multiprocessing_distributed:
            # args.rank in command means node id, in code means global id of model replicas
            args.rank = args.rank * ngpus_per_node + gpu
        # init data parallel training group
        print(f'{args.dist_backend}, {args.dist_url}, {args.world_size}, {args.rank}')
        dist.init_process_group(backend=args.dist_backend, init_method=args.dist_url,
                                world_size=args.world_size, rank=args.rank)
    # 0. create model
    if args.rank == 0:
        print("=> creating model '{}'".format(args.arch))

    if 'cifar10' not in args.data:
        model = models.__dict__[args.arch](num_classes=args.num_classes)
    else:
        model = eval(args.arch+'()')  # build models for CIFAR10

    # 1. put model on one specified gpu or on multi-gpu (DP) in one node or distributed it across nodes (DDP)
    if not torch.cuda.is_available():
        print('this training is using CPU, it will be slow...')
    elif args.distributed:
        if args.gpu is not None:
            torch.cuda.set_device(args.gpu)
            model.cuda(args.gpu)
            # in command, batch-size means total # samples of a batche on one node, workers means the same
            # in code, batch-size means # samples of one batch for each gpu, workers means the same
            args.batch_size = int(args.batch_size / ngpus_per_node)
            args.workers = int(
                (args.workers + ngpus_per_node - 1) / ngpus_per_node)
            print('updated batch_size, workers:',
                  args.batch_size, args.workers)
            model = torch.nn.parallel.DistributedDataParallel(
                model, device_ids=[args.gpu])
            # model = DistributedDataParallel(model, delay_allreduce=True)
            # print(f'*************** rank: {args.rank} ***************')
        else:
            # multi-node, each node has one gpu
            model.cuda()
            model = torch.nn.parallel.DistributedDataParallel(model)
            # model = DistributedDataParallel(model, delay_allreduce=True)
    elif args.gpu is not None:
        # user specify a gpu to train on a single-node multi or single gpu env
        torch.cuda.set_device(args.gpu)
        model = model.cuda(args.gpu)
    else:
        print('======> DP is used.')
        # DataParallel will divide and allocate batch_size to all available GPUs
        if args.arch.startswith('alexnet') or args.arch.startswith('vgg'):
            # FC layers are not DataParallel because it will copy lots of weights to each GPU and takes a lot of time
            model.features = torch.nn.DataParallel(model.features)
            model.cuda()
        else:
            model = torch.nn.DataParallel(model).cuda()

    # 2. define loss function (criterion) and optimizer
    # criterion = nn.CrossEntropyLoss().cuda(args.gpu)
    criterion = nn.CrossEntropyLoss(reduction='none').cuda(args.gpu)
    optimizer = torch.optim.SGD(model.parameters(
    ), args.lr, momentum=args.momentum, weight_decay=args.weight_decay)

    # if args.apex:
    #     # use apex
    #     model, optimizer = amp.initialize(model, optimizer, opt_level="O1")

    # 3. Data loading code
    traindir = os.path.join(args.data, 'train')
    valdir = os.path.join(args.data, 'val')
    normalize = transforms.Normalize(
        mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])

    img_size = 224
    if 'cifar10' in args.data:
        img_size = 32
    print('=> transform img_size:', img_size)
    # train_dataset = datasets.TrainImageFolder(
    train_dataset = dc_datasets.TrainImageFolder(
        traindir,
        transforms.Compose([
            transforms.RandomResizedCrop(
                img_size) if img_size == 224 else transforms.RandomCrop(img_size, padding=4),
            transforms.RandomHorizontalFlip(),
            transforms.ToTensor(),
            normalize,
        ]),
        grpc_port=args.grpc_port
    )
    # train_dataset = datasets.ImageFolder(
    #     traindir,
    #     transforms.Compose([
    #         transforms.RandomResizedCrop(img_size) if img_size==224 else transforms.RandomCrop(img_size, padding=4),
    #         transforms.RandomHorizontalFlip(),
    #         transforms.ToTensor(),
    #         normalize,
    #     ]))

    print('dataset has been initialized')

    if args.distributed:
        # when multi-gpu is used, train_sampler will generate samples different from each other in one epoch
        # train_sampler = torch.utils.data.distributed.DistributedSampler(train_dataset)
        train_sampler = dc_data.distributed.DistributedSampler(train_dataset)
    else:
        # when train in one node with one gpu, train-sampler will defined by dataloader
        train_sampler = None

    if args.cache_type == "quiver":
        # train_loader = torch.utils.data.QuiverDataLoader(
        train_loader = dc_data.QuiverDataLoader(
            train_dataset, batch_size=args.batch_size, shuffle=(
                train_sampler is None),
            num_workers=args.workers, pin_memory=True, sampler=train_sampler, args=args)
    elif args.cache_type == "isa" or args.cache_type == 'isa_lru' or args.sbp:
        # train_loader = torch.utils.data.DataLoader(
        train_loader = dc_data.DataLoader(
            train_dataset, batch_size=args.batch_size, shuffle=(
                train_sampler is None),
            num_workers=args.workers, pin_memory=True, sampler=train_sampler,
            isasampler=True, reuse_factor=args.reuse_factor, beta=args.beta, his_len=args.his_len, warm_up=args.warm_up,
            grpc_port=args.grpc_port, isdistributed=args.distributed, fakecache=args.fakecache)
    else:
        # train_loader = torch.utils.data.DataLoader(
        train_loader = dc_data.DataLoader(
            train_dataset, batch_size=args.batch_size, shuffle=(
                train_sampler is None),
            num_workers=args.workers, pin_memory=True, sampler=train_sampler, grpc_port=args.grpc_port, isdistributed=args.distributed, fakecache=args.fakecache)

    selective_backprop = None
    if args.sbp:
        print('=> Using SBP algorithm...')
        selective_backprop = SelectiveBackPropagation(
            criterion,
            lambda loss: loss.mean().backward(),
            optimizer,
            model,
            args.batch_size,
            len(train_loader),
            loss_selection_threshold=False,
            his_len=args.his_len,
            seed=args.seed,
            warmup=args.warm_up)
        # selective_backprop = SelectiveBackPropagation(
        #                             criterion,
        #                             lambda loss : loss.mean().backward(),
        #                             optimizer,
        #                             model,
        #                             args.batch_size,
        #                             epoch_length=len(train_loader),
        #                             loss_selection_threshold=False)

    print('training dataloader has been initialized')
    # val_loader = torch.utils.data.DataLoader(
    # datasets.ImageFolder(valdir, transforms.Compose([
    val_loader = dc_data.DataLoader(
        dc_datasets.ImageFolder(valdir, transforms.Compose([
            transforms.Resize(
                256) if img_size == 224 else transforms.Resize(32),
            transforms.CenterCrop(img_size),
            transforms.ToTensor(),
            normalize,
        ])),
        batch_size=args.batch_size, shuffle=False,
        num_workers=args.workers, pin_memory=True, validation=True,
        grpc_port=args.grpc_port, isdistributed=False)

    # 4. train epoch by epoch
    for epoch in range(args.start_epoch, args.epochs):
        if args.distributed:
            # set distributed train sampler random seed same as epoch
            train_sampler.set_epoch(epoch)
        # adjust lr
        adjust_learning_rate(optimizer, epoch, args)

        # train for one epoch
        train(train_loader, model, criterion, optimizer,
              epoch, args, selective_backprop)

        # evaluate on validation set

        if args.distributed:
            acc1 = 0  # only see performance on Multiple-GPU, temporarily without caring precision
            time.sleep(30)
        else:
            acc1 = validate(val_loader, model, criterion, args)

        # remember best acc@1 and save checkpoint
        is_best = acc1 > best_acc1
        best_acc1 = max(acc1, best_acc1)

        if not args.multiprocessing_distributed or (args.multiprocessing_distributed
                                                    and args.rank % ngpus_per_node == 0):
            save_checkpoint({
                'epoch': epoch + 1,
                'arch': args.arch,
                'state_dict': model.state_dict(),
                'best_acc1': best_acc1,
                'optimizer': optimizer.state_dict(),
            }, is_best)


def train(train_loader, model, criterion, optimizer, epoch, args, selective_backprop):
    global last_remote_peer_hit, start_count_iter
    batch_time = AverageMeter('Time', ':6.6f')
    data_time = AverageMeter('Data', ':6.6f')
    losses = AverageMeter('Loss', ':.4e')
    top1 = AverageMeter('Acc@1', ':6.2f')
    top5 = AverageMeter('Acc@5', ':6.2f')
    progress = ProgressMeter(
        len(train_loader),
        [batch_time, data_time, losses, top1, top5],
        prefix="Epoch: [{}]".format(epoch))

    # switch to train mode
    model.train()
    if args.distributed:
        dist.barrier()
    end = time.time()
    nhit = 0
    ntotal = 0
    for i, (images, target, imgidx, ishit) in enumerate(train_loader):
        # print('batch size:',len(imgidx))
        # for i, (images, target) in enumerate(train_loader):
        # measure data loading time
        if i>start_count_iter:
            data_time.update(time.time() - end)

        images = images.cuda(args.gpu)
        target = target.cuda(args.gpu)

        # print('target:',target, 'imgidx:', imgidx)

        # compute output and loss for current batch
        if args.sbp:
            loss = torch.Tensor(selective_backprop.reuse_loss(imgidx))
        else:
            output = model(images)
            loss = criterion(output, target)

        # print([loss[i].item() for i in range(args.batch_size)],[i.item() for i in imgidx])

        #
        if args.cache_type == "isa" or args.cache_type == "isa_lru" or args.sbp:
            kvlst = dict(zip([str(j.item()) for j in imgidx], [
                         loss[j].item() for j in range(len(imgidx))]))
            train_loader.update_lst(kvlst)
        # get hit rate
        ntotal += len(images)
        nhit += sum(ishit).item()

        if not args.sbp:
            # measure accuracy and record loss
            acc1, acc5 = accuracy(output, target, topk=(1, 5))
            losses.update(loss.mean().item(), images.size(0))
            top1.update(acc1[0], images.size(0))
            top5.update(acc5[0], images.size(0))

        # compute gradient and do SGD step
        optimizer.zero_grad()
        if args.sbp:
            selective_backprop.selective_back_propagation(loss, images, target, imgidx, beta=args.beta,
                                                          cur_epoch=train_loader.cur_epoch, reuse_factor=args.reuse_factor,
                                                          is_last_batch=(len(imgidx) != args.batch_size))
        else:
            # gradient will be sync in loss.backward()
            loss = loss.mean()
            # if args.apex:
            #     # use apex
            #     with amp.scale_loss(loss, optimizer) as scaled_loss:
            #         scaled_loss.backward()
            # else:
            loss.backward()
        # selective_backprop.selective_back_propagation(loss, images, target)
        optimizer.step()

        # measure elapsed time
        if i>start_count_iter:
            batch_time.update(time.time()-end)

        # st = time.time()
        if (i % args.print_freq == 0) or (i == len(train_loader)-1):
            progress.display(i)
        end = time.time()

    print(f'==> local cache hit rate for current epoch: {nhit}/{ntotal}={round(nhit/ntotal,4)*100}%')
    # print remote peer hit times
    if args.distributed:
        dist.barrier()
    remote_peer_hit = train_loader.get_remote_hit_results()
    this_epoch_remote_peer_hit = remote_peer_hit - last_remote_peer_hit
    print('==> Get remote_hit_results from server:', this_epoch_remote_peer_hit)
    last_remote_peer_hit = remote_peer_hit


def validate(val_loader, model, criterion, args):
    batch_time = AverageMeter('Time', ':6.6f')
    losses = AverageMeter('Loss', ':.4e')
    top1 = AverageMeter('Acc@1', ':6.2f')
    top5 = AverageMeter('Acc@5', ':6.2f')
    progress = ProgressMeter(
        len(val_loader),
        [batch_time, losses, top1, top5],
        prefix='Test: ')

    # switch to evaluate mode
    model.eval()

    with torch.no_grad():
        end = time.time()
        for i, (images, target) in enumerate(val_loader):
            if args.gpu is not None:
                images = images.cuda(args.gpu, non_blocking=True)
            if torch.cuda.is_available():
                target = target.cuda(args.gpu, non_blocking=True)

            # compute output
            output = model(images)
            loss = criterion(output, target)

            # measure accuracy and record loss
            acc1, acc5 = accuracy(output, target, topk=(1, 5))
            losses.update(loss.mean().item(), images.size(0))
            top1.update(acc1[0], images.size(0))
            top5.update(acc5[0], images.size(0))

            # measure elapsed time
            batch_time.update(time.time() - end)
            end = time.time()

            if (i % args.print_freq == 0) or (i == len(val_loader)-1):
                progress.display(i)

        # TODO: this should also be done with the ProgressMeter
        print(' * Acc@1 {top1.avg:.3f} Acc@5 {top5.avg:.3f}'
              .format(top1=top1, top5=top5))

    return top1.avg


def save_checkpoint(state, is_best, filename='checkpoint.pth.tar'):
    torch.save(state, filename)
    if is_best:
        shutil.copyfile(filename, 'model_best.pth.tar')


class AverageMeter(object):
    """Computes and stores the average and current value"""

    def __init__(self, name, fmt=':f'):
        self.name = name
        self.fmt = fmt
        self.reset()

    def reset(self):
        self.val = 0
        self.avg = 0
        self.sum = 0
        self.count = 0

    def update(self, val, n=1):
        self.val = val
        self.sum += val * n
        self.count += n
        self.avg = self.sum / self.count

    def __str__(self):
        fmtstr = '{name} {val' + self.fmt + '} ({avg' + self.fmt + '})'
        return fmtstr.format(**self.__dict__)


class ProgressMeter(object):
    def __init__(self, num_batches, meters, prefix=""):
        self.batch_fmtstr = self._get_batch_fmtstr(num_batches)
        self.meters = meters
        self.prefix = prefix

    def display(self, batch):
        entries = [self.prefix + self.batch_fmtstr.format(batch)]
        entries += [str(meter) for meter in self.meters]
        print('\t'.join(entries))

    def _get_batch_fmtstr(self, num_batches):
        num_digits = len(str(num_batches // 1))
        fmt = '{:' + str(num_digits) + 'd}'
        return '[' + fmt + '/' + fmt.format(num_batches) + ']'


def adjust_learning_rate(optimizer, epoch, args):
    """Sets the learning rate to the initial LR decayed by 10 every 30 epochs"""
    lr = args.lr * (0.1 ** (epoch // 30))  # adjust the params here manually
    for param_group in optimizer.param_groups:
        param_group['lr'] = lr


def accuracy(output, target, topk=(1,)):
    """Computes the accuracy over the k top predictions for the specified values of k"""
    with torch.no_grad():
        maxk = max(topk)
        batch_size = target.size(0)

        _, pred = output.topk(maxk, 1, True, True)
        pred = pred.t()
        correct = pred.eq(target.view(1, -1).expand_as(pred))

        res = []
        for k in topk:
            correct_k = correct[:k].reshape(-1).float().sum(0, keepdim=True)
            res.append(correct_k.mul_(100.0 / batch_size))
        return res


if __name__ == '__main__':
    print('args.req_addr:', args.req_addr)
    # make dir for log placement
    response = requests.get(args.req_addr+"cache")
    info = response.json()

    import mjob_utils
    mjobmetadata = mjob_utils.MultijobMetaData(args.req_addr)
    mjobmetadata.multi_job_init()
    mjobmetadata.multi_job_update_info(sys.argv[0], " ".join(sys.argv[1:]))
    
    main()

    mjobmetadata.multi_job_stop()
    print(mjobmetadata)
