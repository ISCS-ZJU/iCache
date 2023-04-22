import os
import sys
# install packages

# backeup original related torch and torchvision modules
import torch, torchvision
# TODO: for pytorch 1.4
if "1.4" not in torch.__version__:
    print("The current script applies: pytorch 1.4，but current version is", torch.__version__)
    sys.exit(0)
torch_install_path = os.path.dirname(torch.__file__)
torchvision_install_path = os.path.dirname(torchvision.__file__)

# backup files in torch/utils/data, files in torch/utils/data/_utils, files in torchvision/datasets
dirname = ['torch/utils/data', 'torch/utils/data/_utils', 'torchvision/datasets']

for dirname_ in dirname:
    oridirname = dirname_
    dirname_ = dirname_.replace('/','_')
    
    filesname = os.listdir(dirname_)
    for filename in filesname:
        if 'torchvision' in dirname_:
            # torchvision 
            if filename in ["__init__.py"]:
                continue
            if filename.endswith('.py'):
                ori_filename = filename
                if '1.4' in filename:
                    ori_filename = "".join(filename.split("1.4"))
                # print(ori_filename)
                # backup
                fdirpath = os.path.join(torchvision_install_path, oridirname[12:])
                filepath = os.path.join(fdirpath, ori_filename)
                bkpfilepath = os.path.join(fdirpath, ori_filename+'.bkp')
                # fix bug: backup file overwritten
                if not os.path.exists(bkpfilepath):
                    os.system(f"mv {filepath} {bkpfilepath}")
                    print(filepath, bkpfilepath)
                
                
                # ln -sf
                curfilepath = os.path.join(os.path.join(os.getcwd(), dirname_), filename)
                os.system(f'ln -sf {curfilepath} {filepath}')
                print(curfilepath, "->",filepath)
            
        else:
            # torch
            if filename in ["__init__.py"]:
                continue
            if filename.endswith('.py'):
                ori_filename = filename
                if '1.4' in filename:
                    ori_filename = "".join(filename.split("1.4"))
                # print(ori_filename)
                # backup
                fdirpath = os.path.join(torch_install_path, oridirname[6:])
                filepath = os.path.join(fdirpath, ori_filename)
                bkpfilepath = os.path.join(fdirpath, ori_filename+'.bkp')
                if not os.path.exists(bkpfilepath):
                    os.system(f"mv {filepath} {bkpfilepath}")
                    print(filepath, bkpfilepath)
                
                
                # ln -sf
                curfilepath = os.path.join(os.path.join(os.getcwd(), dirname_), filename)
                os.system(f'ln -sf {curfilepath} {filepath}')
                print(curfilepath, "->",filepath)

# ln -sf dcrpc
dcrpcdirname = "dcrpc"
filesname = os.listdir(dcrpcdirname)
for filename in filesname:
    if filename.endswith('.py'):
        fdirpath = os.path.join(torchvision_install_path, "datasets")
        filepath = os.path.join(fdirpath, filename)
        
        # ln -sf
        curfilepath = os.path.join(os.path.join(os.getcwd(), dcrpcdirname), filename)
        os.system(f'ln -sf {curfilepath} {filepath}')
        print(curfilepath, "->",filepath)