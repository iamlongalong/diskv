package dlist

// 直接基于文件的固定长度的链表，修改也是原地修改

// 一个内部的 diskv，用于存储变长的内容，eg: []bytes 等
// [0] => 对应一个 key存的 value
// list 的修改仅重新写固定长度的 block，用来表达指针。