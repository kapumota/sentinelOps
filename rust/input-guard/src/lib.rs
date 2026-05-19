pub fn validate_input(input: &str) -> Result<(), String> {
    let trimmed = input.trim();

    if trimmed.is_empty() {
        return Err("no se permite una entrada vacía".to_string());
    }

    if trimmed.len() > 120 {
        return Err("la entrada supera 120 caracteres".to_string());
    }

    if trimmed.chars().any(|c| c.is_ascii_control()) {
        return Err("control characters are not allowed".to_string());
    }

    let forbidden = ["&&", "||", "../", "$(", "`", ";", "|", ">", "<"];
    for token in forbidden {
        if trimmed.contains(token) {
            return Err(format!("se detectó un token prohibido: {token}"));
        }
    }

    let allowed = trimmed.chars().all(|c| {
        c.is_ascii_alphanumeric() || matches!(c, ' ' | '-' | '_' | '.')
    });

    if !allowed {
        return Err("input contains unsupported characters".to_string());
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::validate_input;

    #[test]
    fn accepts_simple_command() {
        assert!(validate_input("status").is_ok());
    }

    #[test]
    fn accepts_command_with_argument() {
        assert!(validate_input("audit json").is_ok());
    }

    #[test]
    fn rejects_empty_input() {
        assert!(validate_input("   ").is_err());
    }

    #[test]
    fn rejects_forbidden_token() {
        let result = validate_input("status && whoami");
        assert!(result.is_err());
    }

    #[test]
    fn rejects_unsupported_characters() {
        let result = validate_input("status:all");
        assert!(result.is_err());
    }

    #[test]
    fn rejects_too_long_input() {
        let input = "a".repeat(121);
        let result = validate_input(&input);
        assert!(result.is_err());
    }
}
