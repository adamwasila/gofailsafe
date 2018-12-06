# Gofailsafe
*Deadly simple yet powerfull failure handling, now in your favourite language.*

## Introduction

Started as a simple hack to implement [failsafe](https://github.com/jhalterman/failsafe) patterns in Go. It was never meant to copy underlying mechanics or API accurately. Plan is to make something roughly simmilar that will be as fun and easy to use as original Java library.

Implemented:

* Retries
  * fixed delay between retries
  * maximum number of retries
  * user defined retry predicate
  * retry on panic

## Contribute

It is very experimental and early stage project so I don't expect a crowd here. But if you like: please follow documentation and examples, play with the code, try to break it. Then simply propose a change (any change, bugfix or idea) by submitting an issue.

## License

Released under the Apache 2.0 license.
