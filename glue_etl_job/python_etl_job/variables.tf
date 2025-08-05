variable "arguments" {
  type = map(string)
}

variable "name" {
  type = string
}

variable "role_arn" {
  type = string
}

variable "script_bucket" {
  type = string
}

variable "script_key" {
  type = string
}

variable "requirements_key" {
  type    = string
  default = ""
}