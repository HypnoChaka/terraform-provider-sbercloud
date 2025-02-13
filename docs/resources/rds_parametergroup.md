---
subcategory: "Relational Database Service (RDS)"
---

# sbercloud\_rds\_parametergroup

Manages a RDS ParameterGroup resource within SberCloud.

## Example Usage

```hcl
resource "sbercloud_rds_parametergroup" "pg_1" {
  name        = "pg_1"
  datastore {
    type    = "mysql"
    version = "5.6"
  }
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required, String) The parameter group name. It contains a maximum of 64 characters.

* `description` - (Optional, String) The parameter group description. It contains a maximum of 256 characters and cannot contain the following special characters:>!<"&'= the value is left blank by default.

* `values` - (Optional, Map) Parameter group values key/value pairs defined by users based on the default parameter groups.

* `datastore` - (Required, List, ForceNew) Database object. The database object structure is documented below. Changing this creates a new parameter group.

The `datastore` block supports:

* `type` - (Required, String) The DB engine. Currently, MySQL, PostgreSQL, and Microsoft SQL Server are supported. The value is case-insensitive and can be mysql, postgresql, or sqlserver.

* `version` - (Required, String) Specifies the database version.

	* MySQL databases support MySQL 5.6, 5.7 and 8.0. Example value: 5.7.
	* PostgreSQL databases support PostgreSQL 9.5, 9.6, 10, 11 and 12. Example value: 9.5.
	* Microsoft SQL Server databases support 2012_SE, 2014_SE, 2016_SE, 2012_EE, 2014_EE, 2016_EE and 2017_EE. Example value: 2014_SE.


## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` -  ID of the parameter group.

* `configuration_parameters` - Indicates the parameter configuration defined by users based on the default parameters groups.

	* `name` - Indicates the parameter name.
	* `value` - Indicates the parameter value.
	* `restart_required` - Indicates whether a restart is required.
	* `readonly` - Indicates whether the parameter is read-only.
	* `value_range` - Indicates the parameter value range.
	* `type` - Indicates the parameter type.
	* `description` - Indicates the parameter description.

## Timeouts
This resource provides the following timeouts configuration options:
- `create` - Default is 10 minute.
- `delete` - Default is 10 minute.

## Import

Parameter groups can be imported using the `id`, e.g.

```
$ terraform import sbercloud_rds_parametergroup.pg_1 7117d38e-4c8f-4624-a505-bd96b97d024c
```
