class Queue:
    def __init__(self):
        self.items = []

    def isEmpty(self):
        return self.items == []

    def enqueue(self, item):
        self.items.insert(0,item)

    def dequeue(self):
        return self.items.pop()

    def size(self):
        return len(self.items)
    
queue = Queue()

for x in range(4):
    queue.enqueue(x)
    
    
    
a = queue.dequeue()
a = queue.dequeue()
print(a)
    
print(queue.isEmpty())