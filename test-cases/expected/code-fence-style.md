# Code fence style

Tilde fences are rewritten with the configured marker.

```
plain block
```

The info string is kept as written.

```python
print("hi")
```

An over-long fence shrinks back to three characters.

```
already backticks
```

A fence whose content holds a run of the target marker widens instead of colliding with it.

````
Example:

```js
nested();
```
````

A fence that already uses the target marker is left alone.

```
untouched
```

An indented fence keeps its indentation.

- item

  ```sh
  echo hi
  ```

A run of the other marker inside a fence is content, not a fence edge.

```
~~~ not a fence here
```

Indented code blocks are left as they are (MD046 is a separate task):

    indented code block
    stays indented
