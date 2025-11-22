# filter-address-book

Use this filter in an OpenSMTP filter chain after filter-rspamd to apply an
`X-Address-Book` keyword header to each message with a From address contained
in one of the recipient's CardDAV address books.  The value of the header is
set to the name of the address book.
