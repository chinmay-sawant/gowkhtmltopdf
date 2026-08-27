---
name: golang-anti-patterns
description: Comprehensive catalog of the top 50 Go anti-patterns, detection heuristics, and their idiomatic Golang pattern replacements. Covers concurrency, error handling, memory allocation, nil safety, context lifecycle, API ergonomics, and stdlib performance.
---

# Top 50 Golang Anti-Patterns and Idiomatic Patterns Guide

This skill provides a comprehensive reference catalog of the top 50 Go anti-patterns, detection heuristics, flawed code examples, and idiomatic Go pattern replacements. Use this skill to audit, review, refactor, and write robust, high-performance, and idiomatic Go code.

---

## Catalog Overview

| Category | Count | Scope |
| :--- | :--- | :--- |
| **1. Concurrency & Goroutines** | 9 | Goroutine leaks, mutex copying, channel deadlocks, race conditions |
| **2. Error Handling & Sentinels** | 9 | Swallowed errors, %v wrapping, string comparison, panic flow |
| **3. Memory, Slices & Allocation** | 8 | Subslice pinning, append thrashing, pool bounds, defer in loops |
| **4. Types, Interfaces & Nil Safety** | 8 | Typed nil interfaces, interface pollution, mutable globals |
| **5. Context & Cancellation Lifecycle** | 6 | Context in structs, uncancelled timers, missing cancellation checks |
| **6. API Design & Package Seams** | 5 | Boolean blindness, unexported return types, stuttering packages |
| **7. Performance & Stdlib Pitfalls** | 5 | time.After in loops, hot regex compilation, string concat |
| **Total** | **50** | **Complete Codebase Audit Scope** |

---

## Category 1: Concurrency, Synchronization & Goroutine Lifecycle

### Anti-Pattern 1: Goroutine Leak via Unbuffered Channel Receiver Exit
- **Danger:** A spawned worker goroutine sends to an unbuffered channel after the receiver has already returned (e.g. on early error or timeout), permanently blocking the sender and leaking memory.
- **Flawed Code:**
  ```go
  func fetchData(ctx context.Context) (*Data, error) {
      ch := make(chan *Data) // unbuffered
      go func() {
          res := query()
          ch <- res // blocks forever if context times out before this line
      }()
      select {
      case <-ctx.Done():
          return nil, ctx.Err()
      case d := <-ch:
          return d, nil
      }
  }
  ```
- **Idiomatic Pattern:** Use a buffered channel with capacity matching the worker count, or select on `ctx.Done()` during send.
  ```go
  func fetchData(ctx context.Context) (*Data, error) {
      ch := make(chan *Data, 1) // buffered capacity 1 allows goroutine to exit
      go func() {
          ch <- query()
      }()
      select {
      case <-ctx.Done():
          return nil, ctx.Err()
      case d := <-ch:
          return d, nil
      }
  }
  ```

### Anti-Pattern 2: Calling `sync.WaitGroup.Add` Inside the Spawned Goroutine
- **Danger:** `wg.Add(1)` inside the goroutine races with `wg.Wait()`. If `wg.Wait()` executes before the goroutine starts, `Wait` returns prematurely before work starts.
- **Flawed Code:**
  ```go
  for _, item := range items {
      go func(it Item) {
          wg.Add(1) // race: Wait() can execute before this Add()
          defer wg.Done()
          process(it)
      }(item)
  }
  wg.Wait()
  ```
- **Idiomatic Pattern:** Always call `wg.Add` synchronously in the parent goroutine before spawning.
  ```go
  for _, item := range items {
      wg.Add(1)
      go func(it Item) {
          defer wg.Done()
          process(it)
      }(item)
  }
  wg.Wait()
  ```

### Anti-Pattern 3: Passing `sync.Mutex` or `sync.RWMutex` by Value
- **Danger:** Passing a struct containing a mutex by value copies the mutex internal state, resulting in lock contention on an independent lock while the original lock remains untouched.
- **Flawed Code:**
  ```go
  type Cache struct {
      mu   sync.Mutex
      data map[string]string
  }

  func (c Cache) Set(key, val string) { // value receiver copies mu!
      c.mu.Lock()
      defer c.mu.Unlock()
      c.data[key] = val
  }
  ```
- **Idiomatic Pattern:** Always use pointer receivers for structs containing synchronization primitives, or pass explicit pointer `*sync.Mutex`.
  ```go
  func (c *Cache) Set(key, val string) { // pointer receiver shares mu
      c.mu.Lock()
      defer c.mu.Unlock()
      c.data[key] = val
  }
  ```

### Anti-Pattern 4: Holding Locks Across Blocking Network I/O or Heavy Compute
- **Danger:** Holding `sync.Mutex` or `sync.RWMutex` while performing disk I/O, network requests, or heavy JSON decoding starves all concurrent readers and writers.
- **Flawed Code:**
  ```go
  func (s *Server) Handle(req Request) {
      s.mu.Lock()
      defer s.mu.Unlock()
      resp, err := s.client.Do(req.HTTP) // blocking network I/O under lock!
      s.records[req.ID] = resp
  }
  ```
- **Idiomatic Pattern:** Narrow the lock scope strictly to the shared in-memory state mutation.
  ```go
  func (s *Server) Handle(req Request) {
      resp, err := s.client.Do(req.HTTP) // outside lock
      if err != nil {
          return
      }
      s.mu.Lock()
      s.records[req.ID] = resp
      s.mu.Unlock()
  }
  ```

### Anti-Pattern 5: Reading Shared State Without Lock Synchronization
- **Danger:** Assuming scalar reads (like booleans or integers) are atomic on modern CPUs. Data races cause cache incoherency, word tearing, and undefined compiler optimizations.
- **Flawed Code:**
  ```go
  type Worker struct {
      running bool
  }
  func (w *Worker) Stop() { w.running = false }
  func (w *Worker) Run() {
      for w.running { // data race on read
          doWork()
      }
  }
  ```
- **Idiomatic Pattern:** Use `atomic.Bool`, `atomic.Int64`, or a synchronization channel/mutex.
  ```go
  type Worker struct {
      running atomic.Bool
  }
  func (w *Worker) Stop() { w.running.Store(false) }
  func (w *Worker) Run() {
      for w.running.Load() {
          doWork()
      }
  }
  ```

### Anti-Pattern 6: Spinning Wait Loops Without Yield or Channel Notifications
- **Danger:** Busy-waiting loops consume 100% CPU on worker cores while waiting for a condition to become true.
- **Flawed Code:**
  ```go
  for !done.Load() {
      // spin burning CPU cycles
  }
  ```
- **Idiomatic Pattern:** Use channels, `sync.Cond`, or `sync.WaitGroup` to block until signaled.
  ```go
  select {
  case <-doneChan:
      // proceed after event
  case <-ctx.Done():
      return ctx.Err()
  }
  ```

### Anti-Pattern 7: Channel Publish on Closed Channel (Panic on Send)
- **Danger:** Sending on a closed channel causes an immediate, unrecoverable runtime panic.
- **Flawed Code:**
  ```go
  func producer(ch chan int) {
      ch <- 1
      close(ch)
  }
  func anotherProducer(ch chan int) {
      ch <- 2 // panics if producer() closed ch first
  }
  ```
- **Idiomatic Pattern:** Only the sender that owns the channel should close it, or use `sync.Once` for single-producer teardown.
  ```go
  type SafeChannel struct {
      ch   chan int
      once sync.Once
  }
  func (s *SafeChannel) Close() {
      s.once.Do(func() { close(s.ch) })
  }
  ```

### Anti-Pattern 8: RWMutex Read-to-Write Lock Escalation Deadlock
- **Danger:** Attempting to acquire a write lock (`Lock()`) while already holding a read lock (`RLock()`) within the same goroutine causes an instant recursive deadlock.
- **Flawed Code:**
  ```go
  func (c *Cache) GetOrSet(key string) string {
      c.mu.RLock()
      val, ok := c.data[key]
      if !ok {
          c.mu.Lock() // deadlock! RLock is still held
          c.data[key] = generate()
          c.mu.Unlock()
      }
      c.mu.RUnlock()
      return val
  }
  ```
- **Idiomatic Pattern:** Release the read lock before acquiring the write lock, then verify double-checked state.
  ```go
  func (c *Cache) GetOrSet(key string) string {
      c.mu.RLock()
      val, ok := c.data[key]
      c.mu.RUnlock()
      if ok {
          return val
      }

      c.mu.Lock()
      defer c.mu.Unlock()
      if val, ok = c.data[key]; ok { // double check
          return val
      }
      val = generate()
      c.data[key] = val
      return val
  }
  ```

### Anti-Pattern 9: Unbounded Goroutine Spawning Under Load
- **Danger:** Spawning an unthrottled goroutine per incoming request or job exhausts OS threads and memory, leading to OOM crash.
- **Flawed Code:**
  ```go
  for req := range incoming {
      go handle(req) // unbounded: 100k requests spawn 100k goroutines
  }
  ```
- **Idiomatic Pattern:** Throttle concurrency with a worker pool or weighted semaphore (`golang.org/x/sync/semaphore`).
  ```go
  sem := make(chan struct{}, maxWorkers)
  for req := range incoming {
      sem <- struct{}{}
      go func(r Request) {
          defer func() { <-sem }()
          handle(r)
      }(req)
  }
  ```

---

## Category 2: Error Handling, Sentinels & Panics

### Anti-Pattern 10: Swallowing Errors Silently (The Blank Identifier `_ = err`)
- **Danger:** Discarding errors masks network failures, corrupted data, or resource leaks, causing silent corruption downstream.
- **Flawed Code:**
  ```go
  data, _ := os.ReadFile(path) // error ignored
  return parse(data)
  ```
- **Idiomatic Pattern:** Check and handle or propagate every error with context.
  ```go
  data, err := os.ReadFile(path)
  if err != nil {
      return nil, fmt.Errorf("read config file %q: %w", path, err)
  }
  return parse(data)
  ```

### Anti-Pattern 11: Error String Matching via `strings.Contains(err.Error(), ...)`
- **Danger:** Fragile string matching breaks when error message formatting changes or when errors are translated/localized.
- **Flawed Code:**
  ```go
  if strings.Contains(err.Error(), "connection refused") {
      retry()
  }
  ```
- **Idiomatic Pattern:** Use typed sentinel errors with `errors.Is` or custom error types with `errors.As`.
  ```go
  if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, ErrConnectionRefused) {
      retry()
  }
  ```

### Anti-Pattern 12: Wrapping Errors with `%v` or `%s` Instead of `%w`
- **Danger:** Using `%v` or `%s` destroys the error chain, preventing callers from inspecting root cause sentinels via `errors.Is` and `errors.As`.
- **Flawed Code:**
  ```go
  if err != nil {
      return fmt.Errorf("database query failed: %v", err) // %v breaks error unwrapping
  }
  ```
- **Idiomatic Pattern:** Use `%w` to preserve the unwrapping chain for caller inspection.
  ```go
  if err != nil {
      return fmt.Errorf("database query failed: %w", err)
  }
  ```

### Anti-Pattern 13: Redundant Error Wrapping Without Context (`fmt.Errorf("%w", err)`)
- **Danger:** Wrapping an error with only `fmt.Errorf("%w", err)` adds zero contextual information while allocating a new wrapper struct.
- **Flawed Code:**
  ```go
  if err != nil {
      return fmt.Errorf("%w", err) // redundant allocation with zero new context
  }
  ```
- **Idiomatic Pattern:** Return `err` directly, or add descriptive domain context.
  ```go
  if err != nil {
      return fmt.Errorf("fetch user profile for ID %d: %w", userID, err)
  }
  ```

### Anti-Pattern 14: Using Panic for Ordinary Flow Control
- **Danger:** `panic` is reserved for unrecoverable programmer errors (e.g. nil dereferences, invariant violations). Using panic for routine errors degrades performance and violates Go error conventions.
- **Flawed Code:**
  ```go
  func ParseInt(s string) int {
      val, err := strconv.Atoi(s)
      if err != nil {
          panic(err) // bad: panic as flow control
      }
      return val
  }
  ```
- **Idiomatic Pattern:** Return `(T, error)` tuples and let the caller decide how to handle failures.
  ```go
  func ParseInt(s string) (int, error) {
      val, err := strconv.Atoi(s)
      if err != nil {
          return 0, fmt.Errorf("parse integer %q: %w", s, err)
      }
      return val, nil
  }
  ```

### Anti-Pattern 15: Inconsistent Sentinel Error Naming and Prefixes
- **Danger:** Sentinel errors missing the standard `Err` prefix or package namespace cause ambiguity in logs and debugging.
- **Flawed Code:**
  ```go
  var (
      BadInput    = errors.New("bad input") // missing Err prefix
      NotFoundErr = errors.New("not found") // non-standard suffix
  )
  ```
- **Idiomatic Pattern:** Standardize sentinels with `Err` prefix and canonical package prefix in description.
  ```go
  var (
      ErrBadInput = errors.New("mypackage: invalid input argument")
      ErrNotFound = errors.New("mypackage: resource not found")
  )
  ```

### Anti-Pattern 16: Shadowing the `err` Variable in Inner Scopes
- **Danger:** Short variable declaration `:=` inside an `if` or `for` block creates a new inner `err` variable, leaving the outer `err` unassigned and causing callers to observe `nil`.
- **Flawed Code:**
  ```go
  var err error
  if condition {
      result, err := calculate() // creates inner err; outer err remains nil!
      if err != nil {
          return err
      }
      use(result)
  }
  return err // returns nil even if calculation failed!
  ```
- **Idiomatic Pattern:** Assign explicitly to outer variables or return immediately.
  ```go
  if condition {
      result, err := calculate()
      if err != nil {
          return fmt.Errorf("calculate: %w", err)
      }
      use(result)
  }
  return nil
  ```

### Anti-Pattern 17: Panic in Goroutine Without Defer-Recover Isolation
- **Danger:** An unhandled panic in any spawned goroutine crashes the entire Go process, taking down the entire service.
- **Flawed Code:**
  ```go
  go func() {
      // If thirdPartyLibrary.Process() panics, the entire process terminates
      thirdPartyLibrary.Process()
  }()
  ```
- **Idiomatic Pattern:** Install a deferred recovery handler in background worker boundaries.
  ```go
  go func() {
      defer func() {
          if r := recover(); r != nil {
              log.Printf("recovered background panic: %v\n%s", r, debug.Stack())
          }
      }()
      thirdPartyLibrary.Process()
  }()
  ```

### Anti-Pattern 18: Discarding Error from Deferred Calls (`defer file.Close()`)
- **Danger:** Discarding `file.Close()` error on writable files masks write flush failures and data loss.
- **Flawed Code:**
  ```go
  func WriteData(filename string, data []byte) error {
      f, err := os.Create(filename)
      if err != nil {
          return err
      }
      defer f.Close() // write error on close is silently ignored!
      _, err = f.Write(data)
      return err
  }
  ```
- **Idiomatic Pattern:** Capture deferred close errors using named return parameters and `errors.Join`.
  ```go
  func WriteData(filename string, data []byte) (retErr error) {
      f, err := os.Create(filename)
      if err != nil {
          return err
      }
      defer func() {
          if closeErr := f.Close(); closeErr != nil {
              retErr = errors.Join(retErr, fmt.Errorf("close file: %w", closeErr))
          }
      }()
      _, err = f.Write(data)
      return err
  }
  ```

---

## Category 3: Memory, Slices, Buffers & Allocations

### Anti-Pattern 19: Memory Leak via Subslice Pinning Large Underlying Array
- **Danger:** Taking a small slice of a large array or byte slice keeps the entire large underlying array alive in memory, preventing GC reclamation.
- **Flawed Code:**
  ```go
  func getHeader(rawBigData []byte) []byte {
      return rawBigData[:16] // keeps multi-megabyte rawBigData in memory!
  }
  ```
- **Idiomatic Pattern:** Copy the required subslice into a freshly allocated slice.
  ```go
  func getHeader(rawBigData []byte) []byte {
      header := make([]byte, 16)
      copy(header, rawBigData[:16])
      return header
  }
  ```

### Anti-Pattern 20: Slice Append Thrashing Without Capacity Pre-allocation
- **Danger:** Appending in a loop without initial capacity hint causes repeated logarithmic re-allocations and memory copying.
- **Flawed Code:**
  ```go
  var result []Item // cap = 0
  for _, id := range ids {
      result = append(result, loadItem(id)) // triggers reallocations: 1, 2, 4, 8, 16...
  }
  ```
- **Idiomatic Pattern:** Pre-allocate slice with known exact or upper-bound capacity.
  ```go
  result := make([]Item, 0, len(ids))
  for _, id := range ids {
      result = append(result, loadItem(id))
  }
  ```

### Anti-Pattern 21: Storing Unbounded Buffer Sizes in `sync.Pool`
- **Danger:** Returning multi-hundred-megabyte buffers to a `sync.Pool` pins large allocations permanently, causing memory bloat.
- **Flawed Code:**
  ```go
  func PutBuffer(buf *bytes.Buffer) {
      bufPool.Put(buf) // stores 500MB buffer in pool indefinitely
  }
  ```
- **Idiomatic Pattern:** Enforce an upper capacity limit before returning buffers to the pool.
  ```go
  const maxPoolBufferSize = 16 * 1024 * 1024 // 16 MiB max
  func PutBuffer(buf *bytes.Buffer) {
      if buf.Cap() <= maxPoolBufferSize {
          buf.Reset()
          bufPool.Put(buf)
      }
  }
  ```

### Anti-Pattern 22: Accumulating `defer` Statements in Tight Loops
- **Danger:** `defer` executes when the enclosing function returns, not at loop iteration boundaries. Defers inside loops accumulate memory and hold file descriptors open.
- **Flawed Code:**
  ```go
  for _, file := range fileList {
      f, _ := os.Open(file)
      defer f.Close() // does not close until the entire for loop finishes!
      process(f)
  }
  ```
- **Idiomatic Pattern:** Wrap the loop body in a helper function or execute cleanup explicitly.
  ```go
  for _, file := range fileList {
      if err := processFile(file); err != nil {
          return err
      }
  }
  func processFile(name string) error {
      f, err := os.Open(name)
      if err != nil {
          return err
      }
      defer f.Close()
      return process(f)
  }
  ```

### Anti-Pattern 23: Passing Heavy Structs by Value Across Hot Calls
- **Danger:** Passing large structs (>128 bytes) by value copies the entire memory block to the stack on every call, burning CPU cache and bandwidth.
- **Flawed Code:**
  ```go
  type RenderStyle struct {
      // 1.5 KB of style properties
  }
  func computeLayout(style RenderStyle) { // copies 1.5 KB on every call
      // ...
  }
  ```
- **Idiomatic Pattern:** Pass pointers `*RenderStyle` for read-only inspection or mutation of large structs.
  ```go
  func computeLayout(style *RenderStyle) { // passes 8-byte pointer
      // ...
  }
  ```

### Anti-Pattern 24: Unbuffered I/O on Frequent Small Writes
- **Danger:** Direct calls to `io.Writer` (e.g. `os.File` or `net.Conn`) on small byte chunks trigger costly syscalls per write.
- **Flawed Code:**
  ```go
  for _, b := range stream {
      file.Write([]byte{b}) // 1 syscall per byte!
  }
  ```
- **Idiomatic Pattern:** Wrap writes in `bufio.Writer` and flush once upon completion.
  ```go
  bw := bufio.NewWriter(file)
  defer bw.Flush()
  for _, b := range stream {
      bw.WriteByte(b)
  }
  ```

### Anti-Pattern 25: Escaping Slice Pointers via Index Pointer Retention
- **Danger:** Retaining a pointer to an element of a growing slice (`&slice[i]`) causes subtle memory bugs when subsequent `append` reallocates the slice backing array.
- **Flawed Code:**
  ```go
  items := []Item{...}
  ptr := &items[0]
  items = append(items, newItem) // reallocates backing array!
  ptr.Name = "updated" // writes to old discarded backing array
  ```
- **Idiomatic Pattern:** Store slice indices instead of direct pointers to slice elements, or use a slice of pointers `[]*Item`.
  ```go
  items := []*Item{...}
  item := items[0]
  items = append(items, newItem)
  item.Name = "updated" // safely mutates referenced heap object
  ```

### Anti-Pattern 26: Storing Pointers in Arrays of Short-Lived Value Types
- **Danger:** Slices of pointers (`[]*SmallStruct`) incur multiple allocations and pointer chasing during GC traversal, whereas value slices (`[]SmallStruct`) offer contiguous memory cache locality.
- **Flawed Code:**
  ```go
  type Point struct{ X, Y float64 }
  points := make([]*Point, 1000000) // 1M separate heap allocations!
  ```
- **Idiomatic Pattern:** Store small structs by value for contiguous cache-line utilization and zero-allocation slice initialization.
  ```go
  points := make([]Point, 1000000) // 1 single contiguous allocation
  ```

---

## Category 4: Types, Structs, Interfaces & Nil Safety

### Anti-Pattern 27: Returning a Non-Nil Interface Containing a Typed Nil Pointer
- **Danger:** In Go, an interface value is `nil` only if both its type and value are `nil`. Returning a typed `(*MyError)(nil)` as an `error` interface returns a non-nil interface, breaking `if err != nil` checks.
- **Flawed Code:**
  ```go
  func validate() error {
      var customErr *CustomError = nil
      if invalid {
          customErr = &CustomError{"bad"}
      }
      return customErr // returns non-nil interface (type *CustomError, val nil)!
  }
  ```
- **Idiomatic Pattern:** Return untyped literal `nil` explicitly when returning no error.
  ```go
  func validate() error {
      if invalid {
          return &CustomError{"bad"}
      }
      return nil // explicitly nil interface
  }
  ```

### Anti-Pattern 28: Interface Pollution (Premature Interface Extraction)
- **Danger:** Defining 1-method interfaces for every struct before concrete consumer requirements emerge increases indirection, complicates navigation, and violates "Accept interfaces, return structs".
- **Flawed Code:**
  ```go
  type UserServiceInterface interface {
      GetUser(id int) (*User, error)
  }
  type UserService struct{} // only 1 implementation ever exists
  ```
- **Idiomatic Pattern:** Define concrete structs in the producer package; let consumer packages define interfaces at the call site when mocking or abstraction is needed.
  ```go
  type UserService struct{}
  func (s *UserService) GetUser(id int) (*User, error) { ... }
  ```

### Anti-Pattern 29: Mutable Package-Level Global State
- **Danger:** Global variables create hidden coupling across packages, make unit tests order-dependent, and cause data races under concurrent execution.
- **Flawed Code:**
  ```go
  var DefaultClient = http.DefaultClient // global mutable client
  ```
- **Idiomatic Pattern:** Encapsulate dependencies inside structs with constructor injection.
  ```go
  type Service struct {
      client *http.Client
  }
  func NewService(client *http.Client) *Service {
      if client == nil {
          client = http.DefaultClient
      }
      return &Service{client: client}
  }
  ```

### Anti-Pattern 30: Monolithic "Fat" Structs with Disparate Concerns
- **Danger:** Monolithic structs combining unrelated lifecycle concerns (e.g. layout + table formatting + sticky scrolling + paint caches) increase memory footprint and cognitive load.
- **Flawed Code:**
  ```go
  type Box struct {
      // 50 fields covering block, flex, grid, table, sticky, SVG, canvas
  }
  ```
- **Idiomatic Pattern:** Compose focused sub-structures or use contextual extension pointers.
  ```go
  type Box struct {
      Rect      Rectangle
      Style     *ResolvedStyle
      TableMeta *TableMetadata  // allocated only on table elements
      Sticky    *StickyMetadata // allocated only on sticky elements
  }
  ```

### Anti-Pattern 31: Nil Map Write Panic (`m[k] = v` on uninitialized map)
- **Danger:** Reading from a nil map is safe in Go (returns zero value), but writing to a nil map causes an immediate panic.
- **Flawed Code:**
  ```go
  type Registry struct {
      entries map[string]string // uninitialized nil map
  }
  func (r *Registry) Register(k, v string) {
      r.entries[k] = v // panics if r.entries is nil!
  }
  ```
- **Idiomatic Pattern:** Initialize maps in constructors or lazy-initialize on first write.
  ```go
  func (r *Registry) Register(k, v string) {
      if r.entries == nil {
          r.entries = make(map[string]string)
      }
      r.entries[k] = v
  }
  ```

### Anti-Pattern 32: Unchecked Type Assertions (`val.(TargetType)`)
- **Danger:** Directly asserting `val.(ConcreteType)` without the comma-ok idiom panics at runtime if the underlying dynamic type does not match.
- **Flawed Code:**
  ```go
  func process(node any) {
      elem := node.(*ElementNode) // panics if node is *TextNode
  }
  ```
- **Idiomatic Pattern:** Always use comma-ok idiom or type switches.
  ```go
  func process(node any) {
      elem, ok := node.(*ElementNode)
      if !ok {
          return // safely handle non-element nodes
      }
      elem.Render()
  }
  ```

### Anti-Pattern 33: Pointer Indirection for Immutable Primitive Options
- **Danger:** Using `*int` or `*string` for all struct fields forces heap allocations and nil checks everywhere when zero-value defaults suffice.
- **Flawed Code:**
  ```go
  type Config struct {
      Timeout *time.Duration // requires caller to do ptr := 5*time.Second; &ptr
      Retries *int
  }
  ```
- **Idiomatic Pattern:** Use value types for scalar configurations, and reserve pointers strictly for tri-state optional booleans (`*bool`).
  ```go
  type Config struct {
      Timeout time.Duration // 0 means default timeout
      Retries int           // 0 means default retries
      Collate *bool         // nil = default, &true = on, &false = off
  }
  ```

### Anti-Pattern 34: Exposing Unexported Types in Public API Signatures
- **Danger:** Exporting a public method that returns or takes an unexported type prevents external callers from declaring variables of that type.
- **Flawed Code:**
  ```go
  type privateHelper struct{}
  func GetHelper() privateHelper { // callers cannot name this return type!
      return privateHelper{}
  }
  ```
- **Idiomatic Pattern:** Export the return type or return an exported interface.
  ```go
  type Helper struct{}
  func GetHelper() Helper {
      return Helper{}
  }
  ```

---

## Category 5: Context Propagation & Cancellation Lifecycle

### Anti-Pattern 35: Storing `context.Context` Inside a Struct Field
- **Danger:** Contexts represent per-request lifecycles and deadlines. Storing a context in a struct ties the struct to a single request lifecycle and leads to stale cancellation.
- **Flawed Code:**
  ```go
  type Client struct {
      ctx context.Context // anti-pattern: context in struct
  }
  ```
- **Idiomatic Pattern:** Pass `ctx context.Context` explicitly as the first parameter of every entry point method.
  ```go
  type Client struct{}
  func (c *Client) FetchData(ctx context.Context, id string) (*Data, error) {
      // ...
  }
  ```

### Anti-Pattern 36: Leaking `context.WithCancel` / `WithTimeout` Without `defer cancel()`
- **Danger:** Creating child contexts with timeout or cancel without calling `cancel()` retains goroutine timer references in the runtime until timeout expires.
- **Flawed Code:**
  ```go
  func queryWithTimeout(ctx context.Context) error {
      ctxTimeout, _ := context.WithTimeout(ctx, 5*time.Second) // cancel discarded!
      return execute(ctxTimeout)
  }
  ```
- **Idiomatic Pattern:** Always invoke `defer cancel()` immediately after creating a cancellable context.
  ```go
  func queryWithTimeout(ctx context.Context) error {
      ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
      defer cancel()
      return execute(ctxTimeout)
  }
  ```

### Anti-Pattern 37: Passing `nil` Context Instead of `context.TODO()` or `context.Background()`
- **Danger:** Passing `nil` as a `context.Context` panics when downstream libraries invoke `ctx.Done()` or `ctx.Err()`.
- **Flawed Code:**
  ```go
  client.Do(nil, req) // panics on ctx.Done()
  ```
- **Idiomatic Pattern:** Pass `context.Background()` in top-level runners or `context.TODO()` when context plumbing is pending. Validate `if ctx == nil { return ErrNilContext }` at public boundaries.
  ```go
  client.Do(context.Background(), req)
  ```

### Anti-Pattern 38: Ignoring Context Cancellation in Long-Running Loops
- **Danger:** Loops doing heavy CPU calculation or sequential batch processing that fail to poll `ctx.Done()` will ignore user cancellation or HTTP client aborts.
- **Flawed Code:**
  ```go
  func processBatch(ctx context.Context, items []Item) {
      for _, item := range items {
          expensiveWork(item) // runs to completion even if ctx is cancelled!
      }
  }
  ```
- **Idiomatic Pattern:** Check context cancellation between work units.
  ```go
  func processBatch(ctx context.Context, items []Item) error {
      for _, item := range items {
          if err := ctx.Err(); err != nil {
              return err
          }
          expensiveWork(item)
      }
      return nil
  }
  ```

### Anti-Pattern 39: Creating a Disconnected `context.Background()` Inside a Handled Request
- **Danger:** Creating a new `context.Background()` inside an HTTP handler or pipeline stage breaks trace propagation and ignores incoming client cancellations.
- **Flawed Code:**
  ```go
  func HandleRequest(w http.ResponseWriter, r *http.Request) {
      go worker(context.Background()) // disconnected: ignores client disconnect
  }
  ```
- **Idiomatic Pattern:** Derive child contexts from the request context `r.Context()`, or use `context.WithoutCancel(r.Context())` if asynchronous detached completion is explicitly desired.
  ```go
  func HandleRequest(w http.ResponseWriter, r *http.Request) {
      go worker(context.WithoutCancel(r.Context())) // preserves trace metadata
  }
  ```

### Anti-Pattern 40: Blocking on Non-Interruptible System Calls
- **Danger:** System calls like `os.Open` or raw TCP socket reads may hang indefinitely if context cancellation is not monitored via an asynchronous watcher.
- **Flawed Code:**
  ```go
  func ReadFile(ctx context.Context, path string) ([]byte, error) {
      return os.ReadFile(path) // cannot be cancelled if disk hangs
  }
  ```
- **Idiomatic Pattern:** Use an asynchronous context watcher to close the underlying descriptor if context cancels.
  ```go
  func ReadFile(ctx context.Context, path string) ([]byte, error) {
      f, err := os.Open(path)
      if err != nil {
          return nil, err
      }
      done := make(chan struct{})
      defer close(done)
      go func() {
          select {
          case <-ctx.Done():
              _ = f.Close() // force unblock read
          case <-done:
          }
      }()
      return io.ReadAll(f)
  }
  ```

---

## Category 6: API Design, Package Seams & Module Ergonomics

### Anti-Pattern 41: Boolean Blindness in Function Arguments (`doSomething(true, false, true)`)
- **Danger:** Calls with multiple consecutive unnamed boolean arguments are unreadable and easily swapped by mistake.
- **Flawed Code:**
  ```go
  renderDocument(doc, true, false, true) // what do these booleans mean?
  ```
- **Idiomatic Pattern:** Use named option structs or enum/bitmask configuration flags.
  ```go
  type RenderOptions struct {
      Outline        bool
      Grayscale      bool
      SmartShrinking bool
  }
  renderDocument(doc, RenderOptions{
      Outline:        true,
      Grayscale:      false,
      SmartShrinking: true,
  })
  ```

### Anti-Pattern 42: Package Name Stuttering (`user.UserService`, `layout.LayoutEngine`)
- **Danger:** Repeating package names in exported identifier names creates redundant stuttering when imported.
- **Flawed Code:**
  ```go
  package layout
  type LayoutBox struct{} // imported as layout.LayoutBox -> stutter
  ```
- **Idiomatic Pattern:** Name exported types concisely relative to the package namespace.
  ```go
  package layout
  type Box struct{} // imported as layout.Box
  ```

### Anti-Pattern 43: Circular Package Dependencies Resolved via Artificial Glue Packages
- **Danger:** Resolving import cycles by creating empty `common` or `types` packages spreads domain logic and obscures module boundaries.
- **Flawed Code:**
  ```text
  package layout -> package convert -> package layout (cycle!)
  Resolved via: package types (contains every struct dumped together)
  ```
- **Idiomatic Pattern:** Invert dependencies using consumers' interfaces or restructure package DAG along clear pipeline data stages (e.g. `parse` -> `cascade` -> `layout` -> `paint` -> `pdf`).

### Anti-Pattern 44: Mutating Caller-Provided Slices or Maps
- **Danger:** Mutating slices or maps passed into a public API function causes unexpected side effects and race conditions in caller code.
- **Flawed Code:**
  ```go
  func (d *Document) AddPages(pages []Page) {
      d.Pages = pages // directly aliases caller slice!
      d.Pages[0].Zoom = 1.5 // mutates caller memory
  }
  ```
- **Idiomatic Pattern:** Defensively clone user-supplied slices, byte buffers, and maps upon intake.
  ```go
  func (d *Document) AddPages(pages []Page) {
      d.Pages = slices.Clone(pages)
  }
  ```

### Anti-Pattern 45: God Packages (`utils/`, `helpers/`, `common/`)
- **Danger:** Catch-all utility packages attract disconnected helper functions with unclear ownership, violating cohesion and single-responsibility principles.
- **Flawed Code:**
  ```go
  package utils // contains math helpers, string formatting, HTTP fetching, crypto
  ```
- **Idiomatic Pattern:** Group utilities into domain-focused, cohesive packages (e.g. `internal/geometry`, `internal/line`, `internal/load`).

---

## Category 7: Performance, Standard Library & Profiling Pitfalls

### Anti-Pattern 46: Memory Leak via `time.After` in Repeated `select` Loops
- **Danger:** `time.After(d)` allocates a new `time.Timer` that cannot be garbage collected until the duration expires, causing massive memory leaks in high-frequency select loops.
- **Flawed Code:**
  ```go
  for {
      select {
      case msg := <-ch:
          handle(msg)
      case <-time.After(5 * time.Minute): // leaks a new timer on EVERY received msg!
          heartbeat()
      }
  }
  ```
- **Idiomatic Pattern:** Create and reuse a single `time.Timer` or `time.Ticker`, resetting it as needed.
  ```go
  timer := time.NewTimer(5 * time.Minute)
  defer timer.Stop()
  for {
      select {
      case msg := <-ch:
          handle(msg)
          if !timer.Stop() {
              select {
              case <-timer.C:
              default:
              }
          }
          timer.Reset(5 * time.Minute)
      case <-timer.C:
          heartbeat()
          timer.Reset(5 * time.Minute)
      }
  }
  ```

### Anti-Pattern 47: Compiling Regular Expressions in Hot Loops (`regexp.Compile`)
- **Danger:** `regexp.Compile` parses and builds NFA/DFA state machines on every call, imposing massive CPU and memory allocation penalties.
- **Flawed Code:**
  ```go
  func isValidEmail(email string) bool {
      re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`) // compiled on every call!
      return re.MatchString(email)
  }
  ```
- **Idiomatic Pattern:** Pre-compile regular expressions once at package level or in `sync.OnceValue`.
  ```go
  var emailRegexp = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
  func isValidEmail(email string) bool {
      return emailRegexp.MatchString(email)
  }
  ```

### Anti-Pattern 48: Repeated String Concatenation (`+`) in Loops
- **Danger:** Strings in Go are immutable. Using `s += str` in a loop allocates a new string and copies all previous bytes on every iteration ($O(N^2)$ memory copying).
- **Flawed Code:**
  ```go
  var result string
  for _, line := range lines {
      result += line + "\n" // O(N^2) allocations and copies
  }
  ```
- **Idiomatic Pattern:** Use `strings.Builder` with `Grow` capacity hint for $O(N)$ single-buffer construction.
  ```go
  var b strings.Builder
  for _, line := range lines {
      b.WriteString(line)
      b.WriteByte('\n')
  }
  return b.String()
  ```

### Anti-Pattern 49: Heavy Runtime Reflection in Core Processing Pipelines
- **Danger:** `reflect.ValueOf`, `reflect.TypeOf`, and interface boxing in hot inner loops bypass compiler optimizations, trigger heap allocations, and reduce throughput.
- **Flawed Code:**
  ```go
  func getProperty(target any, prop string) any {
      val := reflect.ValueOf(target)
      return val.FieldByName(prop).Interface() // slow runtime reflection
  }
  ```
- **Idiomatic Pattern:** Use Go 1.18+ generic type parameters, closures, or static lookup dispatch tables.
  ```go
  type accessor[T any] func(*T) string
  var dispatchTable = map[string]accessor[Config]{
      "title": func(c *Config) string { return c.Title },
  }
  ```

### Anti-Pattern 50: Ignoring `io.Closer` in HTTP Response Bodies
- **Danger:** Failing to read remaining response body and close `resp.Body` prevents HTTP/1.1 and HTTP/2 underlying TCP connections from being reused by `http.Transport` connection pooling.
- **Flawed Code:**
  ```go
  resp, err := http.Get(url)
  if err != nil {
      return err
  }
  // resp.Body is not closed, TCP connection is abandoned and leaked!
  ```
- **Idiomatic Pattern:** Always drain remaining bytes and defer close on response bodies.
  ```go
  resp, err := http.Get(url)
  if err != nil {
      return err
  }
  defer func() {
      _, _ = io.Copy(io.Discard, resp.Body)
      _ = resp.Body.Close()
  }()
  ```

---

## Multi-Agent Review Prompt Integration

When conducting a codebase audit using this skill, dispatch specialized reviewers referencing these anti-pattern IDs:
- **Track 1: Concurrency & Lifecycle**: Detect AP-01 through AP-09, AP-35 through AP-40.
- **Track 2: Error Handling & Correctness**: Detect AP-10 through AP-18, AP-27 through AP-34.
- **Track 3: Memory & Allocations**: Detect AP-19 through AP-26, AP-46 through AP-50.
- **Track 4: API & Architecture Ergonomics**: Detect AP-41 through AP-45.

Every reported finding must cite:
1. `AP-XX` identifier and rule name.
2. Exact `file:line` citation.
3. Current flawed code snippet.
4. Idiomatic Go replacement snippet.
