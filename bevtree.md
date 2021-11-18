# 定义
```
Node:
    // Is node running.
    BOOL IsRunning()

    // Run the node.
    Result Run(NodeQueue)

    // Stop the node.
    Stop(NodeQueue)

    // Listen for child's completion evet.
    Result OnChildCompleted(Node, Result, NodeQueue)


Tree:
    Result Run()
```

# workflow

## 定义
行为树的运行，其实就是一棵树的遍历过程。工作流如何设计，取决于行为树以何种方式驱动，是 Tick/Update 形式逐帧驱动，还是以 事件/消息 的方式驱动。

在我定义的这例纯以事件（消息）驱动的行为树中，每次遍历，要么直接得到一个确定的结果（Succes/Failure），标示行为树单次遍历完毕；要么得到行为树正在运行中（Running），表示行为树正在运行异步行为，待行为完成后，会发送事件通知行为树，行为树根据行为的完成结果继续驱动本次行为树的运行。

根据定义，结合树形结构的深度优先遍历，可以得到该种行为树的单次运行流程大致如下：
1. 从根节点出发，依据行为树决策，寻找可执行的行为。
2. 找到行为并运行，若为异步行为，挂起行为树，待行为执行完成，通知行为树运行结果，恢复树的运行，继续步骤3
3. 向上提交结果，依据决策寻找下一个可执行行为。若找到，跳至步骤2
4. 行为树单次运行完毕，返回最终结果

但是，在我实现的工作流中，树的遍历并不完全是深度优先的，可能因为决策的设计，临时切换为广度优先。因为，在我的工作流中，为了优化性能，避免节点的嵌套运行，引入了工作节点队列。从根节点开始，每个节点运行时，依据决策将后续节点推送到工作队列中。工作流依次从工作节点队列中取出节点并运行，直到工作节点为空，即代表运行完成或挂起。可知，并行节点（Parallel）决策时，会将所有子节点推入工作队列，从运行其第一个子节点开始，行为树的遍历变为广度优先，这也正好符合并行节点的定义。

## 实现
根据定义，有两种情况驱动行为树的工作流：
1. 从根节点开始遍历行为树
2. 收到异步行为完成结果

结合两种情况，可认为是从某个特定节点出发，向上或向下进行决策的过程。下面是伪代码：
```
// 起始节点
Node

// 起始方向因子
// Running 代表从 node 开始向下进行决策
// 其它，代表从 node 开始向上进行决策
Result DirF

// 工作节点队列
NodeQueue

// 结果
R = DifF

// 工作节点不为空
LOOP Node != NULL THEN
    IF R != Running THEN
        // 向上
        IF Node.Parent() == NULL THEN
            // 行至根节点，完成
            Node = NULL
        ELSE IF (R = Node.Parent().onChildCompleted(Node, R, NodeQueue)) == Running THEN
            // 向上提交结果后，行为树仍处运行状态，继续执行后续工作节点
            Node = NodeQueue.Pop()
        ELSE THEN
            // 向上提交结果后，父级完成，逐层提交
            Node = Node.Parent() 
    ELSE THEN
        // 向下
        IF Node.IsRunning() THEN
            ERROR "node already running"
        IF (R = Node.Run(NodeQueue)) == Running THEN
            // 运行中，继续执行后续工作节点
            Node = NodeQueue.Pop()

```

```
// ForeNodes 队列中存储当前所有需要运行的并行行为节点
// BackNodes 队列中存储所有下一次需要运行的并行行为节点

// 1. 运行所有行为节点, 若行为未完成，推入到到 backNodes 中的
// 2. 根据所有已完成的节点结果，逐个进行决策，根据决策结果处理决策返回的节点。
//    若父节点完成，停止运行父节点返回的节点，并向上提交结果，知道抵达根节点或父节点未完成；否则，运行返回的子节点，根据结果再次进行决策

LOOP Node = ForeNodes.Pop(); Node != NULL; Node = ForeNodes.Pop() THEN
    IF Node == NULL || Node.IsStopped() CONTINUE

    IF (Result = Node.Run(NodeQue)) == Running THEN
        backNodes.Push(Node)
    ELSE THEN
        LOOP Parent = Node.Parent(); Parent != NULL; Node = Parent, Parent = Parent.Parent() THEN
            Result = Parent.OnChildCompleted(Node, Result, NodeQue)
            LOOP Node = NodeQue.Pop(); Node != NULL; Node = NodeQue.Pop() THEN
                IF Result == Running THEN
                    ForeNodes.Push(node)
                ELSE THEN
                    Node.Stop()
                END
            END   
        END
    END
END

NodeQue = New()
LOOP CompNode = CompletedNodes.Pop(); CompNode != NULL; CompNode = CompletedNodes.Pop() THEN
    Result = CompNode.Result
    Node = CompNode.Node
    LOOP Parent = Node.Parent(); Parent != NULL; Node = Parent, Parent = Parent.Parent() THEN
        Result = Parent.OnChildCompleted(Node, Result, NodeQue)
        IF Result == Running THEN
            LOOP Node = NodeQue.Pop(); Node != NULL; Node = NodeQue.Pop()
                IF (R = Node.Run(NodeQue)) != Running THEN
                    CompletedNodes.Push(Node, R)
                ELSE IF !Node.HasChild() THEN
                    backNodes.Push(Node)
                Node.Stop()
            END
        ELSE THEN
            LOOP Node = NodeQue.Pop(); Node != NULL; Node = NodeQue.Pop()
                Node.Stop()
            END
        END   
    END
END
        
Tmp = ForeNodes
ForeNodes = BackNodes
BackNodes = ForeNodes

```

# TODO List
- [ ] 调整结构，开放自定义节点创建功能（用户自定义节点并在行为树系统中注册）
- [ ] 行为树系统可以注册用户自定义行为