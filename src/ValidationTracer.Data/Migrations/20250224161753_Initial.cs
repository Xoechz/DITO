using Microsoft.EntityFrameworkCore.Migrations;

#nullable disable

namespace ValidationTracer.Data.Migrations
{
    /// <inheritdoc />
    public partial class Initial : Migration
    {
        /// <inheritdoc />
        protected override void Up(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.CreateTable(
                name: "CostCenters",
                columns: table => new
                {
                    Code = table.Column<string>(type: "TEXT", maxLength: 4, nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_CostCenters", x => x.Code);
                });

            migrationBuilder.CreateTable(
                name: "Users",
                columns: table => new
                {
                    EmailAddress = table.Column<string>(type: "TEXT", maxLength: 50, nullable: false),
                    ExternalProperty = table.Column<string>(type: "TEXT", maxLength: 10, nullable: true),
                    InternalProperty = table.Column<string>(type: "TEXT", maxLength: 10, nullable: true),
                    CostCenterCode = table.Column<string>(type: "TEXT", maxLength: 4, nullable: true)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Users", x => x.EmailAddress);
                    table.ForeignKey(
                        name: "FK_Users_CostCenters_CostCenterCode",
                        column: x => x.CostCenterCode,
                        principalTable: "CostCenters",
                        principalColumn: "Code");
                });

            migrationBuilder.CreateIndex(
                name: "IX_Users_CostCenterCode",
                table: "Users",
                column: "CostCenterCode");
        }

        /// <inheritdoc />
        protected override void Down(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.DropTable(
                name: "Users");

            migrationBuilder.DropTable(
                name: "CostCenters");
        }
    }
}
