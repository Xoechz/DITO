using System.ComponentModel.DataAnnotations;

namespace ValidationTracer.Data.Entities;

public class CostCenter
{
    [Key, MaxLength(4)]
    public required string Code { get; set; }
}