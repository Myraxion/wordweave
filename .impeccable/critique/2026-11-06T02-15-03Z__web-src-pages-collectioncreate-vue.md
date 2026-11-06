---
target: 收录与建组页面
total_score: 25
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 2
target_identity: "file:D:\\Workspace\\code\\wordweave\\web\\src\\pages\\CollectionCreate.vue"
target_fingerprint: "sha256:05b282e74027724e2a8a5dfd13c3b53e6631570b1587c33f45e540ab1155a91a"
target_path: "D:\\Workspace\\code\\wordweave\\web\\src\\pages\\CollectionCreate.vue"
timestamp: 2026-11-06T02-15-03Z
slug: web-src-pages-collectioncreate-vue
---
Method: dual-agent (A: /root/design_review · B: /root/detector_review)

# 收录与建组页面评估

目标：web/src/pages/CollectionCreate.vue，/entries/new 与 /groups/new。桌面1440×900、手机390×844，实际Playwright Chromium截图与交互；仅隔离预览数据。

## 设计判断

符合个人英文学习者收录后辨析的任务，绿色身份与中文领域术语一致；主要问题在提交行为、保存边界和长表单操作距离，无需重做视觉风格。

## 启发式评分

| 启发式 | 分数/4 | 依据 |
|---|---:|---|
| 状态可见 | 3 | 收录成功仅称已选择 |
| 符合现实 | 3 | 任务自然、取消范围模糊 |
| 用户控制 | 3 | 有清空、移除及确认 |
| 一致性 | 2 | 搜索Enter提交整个组 |
| 防错 | 2 | 条件校验有效但会误提交 |
| 识别胜于记忆 | 2 | 成员可见，键盘入口缺说明 |
| 效率 | 2 | 连续收词好、建组完成远 |
| 简洁层级 | 3 | 双栏明确、任务长度失衡 |
| 错误恢复 | 3 | 保留输入、取消后果不清楚 |
| 帮助 | 2 | 缺选词键盘提示及完成出口 |
| 总分 | 25/40 | Acceptable |

## 优先问题

1. [P1] 成员搜索Enter意外保存组。实测选affect/effect及关系类型后，在member-search按Enter，直接创建组并跳/groups/3。搜索框应独立处理Enter，只用明确保存动作提交。建议 impeccable harden。
2. [P1] 已保存词条与未保存组边界不清。收录成功显示已选择，取消仅询问有未保存的内容。实测确认后词条保留，数据行为正确但文案令人担忧。成功明确已收录，取消明确仅丢组草稿、词条保留，提供清晰结束收录出口。建议 impeccable clarify。
3. [P2] 手机保存取消在长表单末端。390×844时文档y约1957，桌面也在y1441；操作距离长，无横向溢出。压缩重复说明与空白，探索就近任务动作区、已选数和缺项提示；保留单列顺序。建议 impeccable adapt。
4. [P2] 键盘候选入口无提示。UL可聚焦，checkbox初始tabindex=-1，需要Enter/空格进入、Esc退出，但页面没有说明。增加可见/可访问提示或采用熟悉焦点模式。建议 impeccable harden。
5. [P2] 桌面双栏长度失衡。左栏结束后右栏仍延伸至保存，左侧大片空白。保留确认的双栏，压紧右栏选成员、类型说明和动作，将重复收录入口弱化。建议 impeccable layout。

## 优点

独立收录持久化且取消组后保留词条；连续收录清空并返回拼写焦点；跨搜索保留成员；保存禁用时提供首要缺项提示。

## 认知负担与情绪

负担来自两个保存边界，不是功能太多。七种关系类型采用原生下拉合理，无需机械拆分。进入平静、连续收词反馈顺畅；低谷在已收录后又遇未保存警告，以及手机最后寻找保存。

## Persona

Jordan：误解已选择和笼统取消确认。Sam：候选入口不易发现，搜索回车可能误提交。Casey：成员与说明间滚动多，完成操作太远。

## 次要观察

手机主按钮约41.19px，适配时可提升至约44px；两宽度无横向溢出。groups/new手机会定位建组搜索区，不应称每次必须滚过收录区。未测真实软键盘或屏幕阅读器。

## 检测证据

三个Vue目标CLI均[]，0项。真实运行时注入detect.js后console报告桌面7、手机10项，但无规则及定位，无法逐项确认或判断误报，未计入优先问题。headless无用户可见Human标签页。内置浏览器不可用，采用宿主Playwright fallback。

截图：.scratch/critique-a-desktop.png、.scratch/critique-a-mobile.png、.scratch/collection-critique/b-desktop.png、.scratch/collection-critique/b-mobile.png。

## 待讨论

优先修误提交和保存取消语义，还是手机完成路径？范围先覆盖两个P1，还是五项一起规划？
