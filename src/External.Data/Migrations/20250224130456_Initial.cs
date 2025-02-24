using Microsoft.EntityFrameworkCore.Migrations;

#nullable disable

namespace External.Data.Migrations
{
    /// <inheritdoc />
    public partial class Initial : Migration
    {
        /// <inheritdoc />
        protected override void Up(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.CreateTable(
                name: "ExternalUsers",
                columns: table => new
                {
                    EmailAddress = table.Column<string>(type: "TEXT", maxLength: 50, nullable: false),
                    ExternalProperty = table.Column<string>(type: "TEXT", maxLength: 10, nullable: true),
                    CostCenterCode = table.Column<string>(type: "TEXT", maxLength: 4, nullable: true)
                },
                constraints: table =>
                    table.PrimaryKey("PK_ExternalUsers", x => x.EmailAddress));
        }

        /// <inheritdoc />
        protected override void Down(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.DropTable(
                name: "ExternalUsers");
        }
    }
}