import os

def process():
    for root, _, files in os.walk('.'):
        for file in files:
            if file.endswith('.go') or file == 'go.mod':
                path = os.path.join(root, file)
                with open(path, 'r') as f:
                    content = f.read()
                
                if 'github.com/abel-gezahegn/ledger' in content:
                    content = content.replace('github.com/abel-gezahegn/ledger', 'github.com/rebira678/ledger')
                    with open(path, 'w') as f:
                        f.write(content)

if __name__ == '__main__':
    process()
