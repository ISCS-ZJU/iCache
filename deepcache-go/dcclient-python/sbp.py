'''
ref: https://github.com/Manuscrit/SelectiveBackPropagation
'''

import collections

import numpy as np
import torch
import time
# from loguru import logger


class SelectiveBackPropagation:
    """
    Selective_Backpropagation from paper Accelerating Deep Learning by Focusing on the Biggest Losers
    https://arxiv.org/abs/1910.00762v1
    Without:
            ...
            criterion = nn.CrossEntropyLoss(reduction='none')
            ...
            for x, y in data_loader:
                ...
                y_pred = model(x)
                loss = criterion(y_pred, y).mean()
                loss.backward()
                ...
    With:
            ...
            criterion = nn.CrossEntropyLoss(reduction='none')
            selective_backprop = SelectiveBackPropagation(
                                    criterion,
                                    lambda loss : loss.mean().backward(),
                                    optimizer,
                                    model,
                                    batch_size,
                                    epoch_length=len(data_loader),
                                    loss_selection_threshold=False,
                                    his_len=1024,
                                    seed = args.seed)
            ...
            for x, y in data_loader:
                ...
                with torch.no_grad():
                    y_pred = model(x)
                not_reduced_loss = criterion(y_pred, y)
                selective_backprop.selective_back_propagation(not_reduced_loss, x, y, beta=0.33)
                ...
    """
    def __init__(self, compute_losses_func, update_weights_func, optimizer, model,
                 batch_size, epoch_length, loss_selection_threshold=False, his_len=1024, seed=2022, warmup=3):
        print('extended sbp')
        """
        Usage:
        ```
        criterion = nn.CrossEntropyLoss(reduction='none')
        selective_backprop = SelectiveBackPropagation(
                                    criterion,
                                    lambda loss : loss.mean().backward(),
                                    optimizer,
                                    model,
                                    batch_size,
                                    epoch_length=len(data_loader),
                                    loss_selection_threshold=False,
                                    his_len=1024,
                                    seed=args.seed)
        ```
        :param compute_losses_func: the loss function which output a tensor of dim [batch_size] (no reduction to apply).
        Example: `compute_losses_func = nn.CrossEntropyLoss(reduction='none')`
        :param update_weights_func: the reduction of the loss and backpropagation. Example: `update_weights_func =
        lambda loss : loss.mean().backward()`
        :param optimizer: your optimizer object
        :param model: your model object
        :param batch_size: number of images per batch
        :param loss_selection_threshold: default to False. Set to a float value to select all images with with loss
        higher than loss_selection_threshold. Do not change behavior for loss below loss_selection_threshold.
        """

        self.loss_selection_threshold = loss_selection_threshold
        self.compute_losses_func = compute_losses_func
        self.update_weights_func = update_weights_func
        self.batch_size = batch_size
        self.optimizer = optimizer
        self.model = model
        self.his_len = his_len

        self.loss_hist = collections.deque([], maxlen=self.his_len)
        self.selected_inputs, self.selected_targets = [], []

        # for reuse loss
        self.samples_loss = [0] * epoch_length * batch_size

        # record imgidx used to backprop
        self.selected_imgidx = []

        # self.warmup
        self.warmup_imgs = 0
        self.warmup_epochs = warmup
        self.epochs_img = epoch_length * batch_size
        self.warmup_total_imgs = warmup * self.epochs_img

        # for reproducibility
        if seed is not None:
            np.random.seed(seed)

    def selective_back_propagation(self, loss_per_img, data, targets, imgidx, beta=1, cur_epoch=0, reuse_factor=3, is_last_batch=False):
        effective_batch_loss = None
        all_select = False
        
        # # original IS: uncomments the following block and contents of "if else"
        # if self.warmup_imgs <= self.warmup_total_imgs or (cur_epoch % reuse_factor)==0:
        #     # forward
        #     self.model.train()
        #     output = self.model(data)
        #     loss_per_img = self.compute_losses_func(output, targets)
        #     self.update_loss(imgidx, loss_per_img)
        #     all_select = True
        #     self.warmup_imgs += self.batch_size
        # else:
        #     cpu_losses = loss_per_img.detach().clone().cpu()
        #     self.loss_hist.extend(cpu_losses.tolist())
        #     np_cpu_losses = cpu_losses.numpy()
        #     selection_probabilities = self._get_selection_probabilities(np_cpu_losses, beta)
        #     selection = selection_probabilities > np.random.random(*selection_probabilities.shape)
        
        # # new IS；
        # all forward
        self.model.train()
        output = self.model(data)
        loss_per_img = self.compute_losses_func(output, targets)
        self.update_loss(imgidx, loss_per_img)
        all_select = True
        self.warmup_imgs += self.batch_size

        if all_select:
            self.update_weights_func(loss_per_img)
            effective_batch_loss = loss_per_img
        else:
            # selected_losses = []
            for idx in np.argwhere(selection).flatten():
                # selected_losses.append(np_cpu_losses[idx])
                self.selected_inputs.append(data[idx, ...].detach().clone())
                self.selected_targets.append(targets[idx, ...].detach().clone())
                # self.selected_imgidx.append(imgidx[idx].item())
                if len(self.selected_targets) == self.batch_size:
                    # print('backpropagation imgidx:',self.selected_imgidx)
                    self.model.train()
                    predictions = self.model(torch.stack(self.selected_inputs))
                    effective_batch_loss = self.compute_losses_func(predictions,
                                                                    torch.stack(self.selected_targets))
                    self.update_weights_func(effective_batch_loss)
                    effective_batch_loss = effective_batch_loss.mean()
                    self.model.eval()
                    self.selected_inputs = []
                    self.selected_targets = []
                    # self.selected_imgidx = []
            if is_last_batch and len(self.selected_inputs):
                self.model.train()
                predictions = self.model(torch.stack(self.selected_inputs))
                effective_batch_loss = self.compute_losses_func(predictions,
                                                                torch.stack(self.selected_targets))
                # print('loss.mean:', effective_batch_loss.mean())
                self.update_weights_func(effective_batch_loss)
                # print('last elem of backward:', imgidx[-1])
                effective_batch_loss = effective_batch_loss.mean()
                self.model.eval()
                self.selected_inputs = []
                self.selected_targets = []
            # logger.info("Mean of input loss {}".format(np.array(np_cpu_losses).mean()))
            # logger.info("Mean of loss history {}".format(np.array(self.loss_hist).mean()))
            # logger.info("Mean of selected loss {}".format(np.array(selected_losses).mean()))
            # logger.info("Mean of effective_batch_loss {}".format(effective_batch_loss))
        return effective_batch_loss

    def _get_selection_probabilities(self, loss, beta):
        percentiles = self._percentiles(self.loss_hist, loss)
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
    
    # for reuse history loss
    def update_loss(self, imgidx, not_reduced_loss):
        for i,l in zip(imgidx, not_reduced_loss):
            self.samples_loss[i] = l
    
    def reuse_loss(self, imgidx):
        return [self.samples_loss[i] for i in imgidx]


