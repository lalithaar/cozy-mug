---
title: "form package"
description: "Request binding and validation."
---

`form.Bind[T](c)` validates struct tags such as `required`, `min`, `max`,
and `email`. Values are repopulated with `.Raw("field")`. File fields of type
`*multipart.FileHeader` are auto-bound in multipart requests. Implement custom
validation with a `Validate() []core.FieldError` method.
