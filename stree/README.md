# Subject Tree

Extracted from the NATS server for use as a standalone data structure. See sublist for more information.

Opus 4.6 on the difference between the sublist and subject tree:

Your understanding of the sublist is correct. It’s a pattern-matching trie: you insert keys that may contain wildcards, and you query with a literal key to find all matching patterns. The traversal at each level has to fan out to check both the exact token match and any wildcard nodes. This is a well-understood problem — it’s essentially what MQTT brokers and pub/sub systems all need.

The subject tree (ART) solves a different problem: it’s a key-value store where all keys are literal (no wildcards in stored keys). You insert "foo.bar.baz" → some value and later look up "foo.bar.baz" to get that value back. It also supports a Match(filter) operation where the query contains wildcards — the reverse direction from the sublist.

So what does ART give you over naive approaches?

vs. a hashmap: A hashmap gives you O(1) point lookups, but it can’t do prefix iteration, ordered traversal, or wildcard matching without scanning every key. If you need “give me all keys matching foo.*.baz”, a hashmap has to check every entry. An ART can prune entire subtrees that don’t match, because keys sharing a prefix share a path in the tree.

vs. a basic trie/radix tree: A plain radix tree wastes enormous memory. If you key on bytes, every internal node needs space for up to 256 child pointers — most of which are null. ART’s core insight is to use four different node sizes (4, 16, 48, 256 children) and promote/demote nodes as children are added or removed. A node with 3 children uses a tiny 4-slot node; a node with 200 children uses the full 256-slot array. This makes it competitive with hashmaps on memory while preserving the structural advantages of a trie.

Path compression is the other big win. If you have a million keys that all start with events.user.account., a naive trie stores that shared prefix as a chain of single-child nodes you traverse one by one. ART collapses those into a single node storing the compressed prefix bytes, so traversal skips over them in one comparison rather than walking node-by-node.

Lazy expansion means ART doesn’t even split nodes until it needs to distinguish two keys. If you insert "foo.bar.baz" and nothing else shares that prefix, the tree stores it as a single leaf with the full key — no internal structure at all. The trie only elaborates when a second key forces a fork point.

So to summarize: ART is interesting as a general-purpose ordered key-value structure that gets close to hashmap speed for point lookups, close to hashmap memory efficiency (unlike classical tries), but also supports prefix scans, range queries, and wildcard matching by structure — things hashmaps fundamentally can’t do without brute force.​​​​​​​​​​​​​​​​
