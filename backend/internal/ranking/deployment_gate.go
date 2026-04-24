package ranking

// DeploymentGate enforces a minimum AUC improvement before allowing a new
// model to replace the current deployed version.
type DeploymentGate struct {
	minImprovementFraction float64 // e.g. 0.02 for 2%
}

// NewDeploymentGate creates a gate with the given minimum improvement fraction.
// minImprovement=0.02 means newAUC must be >= currentAUC * 1.02 to pass.
func NewDeploymentGate(minImprovement float64) *DeploymentGate {
	return &DeploymentGate{minImprovementFraction: minImprovement}
}

// MinImprovement returns the configured minimum improvement fraction.
func (g *DeploymentGate) MinImprovement() float64 { return g.minImprovementFraction }

// ShouldDeploy returns true when newAUC meets the improvement threshold over
// currentAUC. When currentAUC is 0 (no deployed model), any positive newAUC passes.
func (g *DeploymentGate) ShouldDeploy(currentAUC, newAUC float64) bool {
	if currentAUC <= 0 {
		return newAUC > 0
	}
	return newAUC >= currentAUC*(1+g.minImprovementFraction)
}
