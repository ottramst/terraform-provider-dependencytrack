# Fetch the latest portfolio-wide metrics snapshot
data "dependencytrack_portfolio_metrics" "current" {}

output "portfolio_risk_score" {
  value = data.dependencytrack_portfolio_metrics.current.inherited_risk_score
}
