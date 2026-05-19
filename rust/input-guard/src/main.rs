use std::{env, process};

use input_guard::validate_input;

fn main() {
    let input = env::args().skip(1).collect::<Vec<String>>().join(" ");

    if input.trim().is_empty() {
        eprintln!("missing input");
        process::exit(2);
    }

    match validate_input(&input) {
        Ok(_) => process::exit(0),
        Err(message) => {
            eprintln!("{message}");
            process::exit(10);
        }
    }
}
