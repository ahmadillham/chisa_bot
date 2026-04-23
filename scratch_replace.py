import os
import re

def process_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    original = content
    
    # Simple replacements
    content = content.replace('"log"\n', '"log/slog"\n')
    
    # Replace log.Printf("[module] error...: %v", err)
    # Regex to capture the format string and the args
    def replacer(match):
        fmt_str = match.group(1)
        args = match.group(2)
        
        # Check if it's an error log
        if 'err' in args.lower() or 'fail' in fmt_str.lower() or 'error' in fmt_str.lower():
            # Replace %v or %s in fmt_str
            clean_fmt = re.sub(r'\[.*?\]\s*', '', fmt_str) # remove [module]
            clean_fmt = re.sub(r':\s*%[vs]', '', clean_fmt)
            clean_fmt = re.sub(r'\s*%[vs]', '', clean_fmt)
            
            # split args
            args_list = [a.strip() for a in args.split(',')]
            slog_args = []
            for a in args_list:
                if a == 'err':
                    slog_args.append('"error", err')
                else:
                    slog_args.append(f'"val", {a}')
            
            return f'slog.Error("{clean_fmt}", {", ".join(slog_args)})'
        else:
            clean_fmt = re.sub(r'\[.*?\]\s*', '', fmt_str)
            clean_fmt = re.sub(r':\s*%[vs]', '', clean_fmt)
            clean_fmt = re.sub(r'\s*%[vs]', '', clean_fmt)
            
            args_list = [a.strip() for a in args.split(',')]
            slog_args = []
            for a in args_list:
                slog_args.append(f'"val", {a}')
                
            return f'slog.Info("{clean_fmt}", {", ".join(slog_args)})'

    content = re.sub(r'log\.Printf\("(.*?)"\s*,\s*(.*?)\)', replacer, content)
    
    # Replace log.Printf without args (if any)
    def replacer_no_args(match):
        fmt_str = match.group(1)
        clean_fmt = re.sub(r'\[.*?\]\s*', '', fmt_str)
        return f'slog.Info("{clean_fmt}")'
        
    content = re.sub(r'log\.Printf\("(.*?)"\)', replacer_no_args, content)
    
    # Replace log.Println
    content = re.sub(r'log\.Println\((.*?)\)', r'slog.Info(\1)', content)
    
    # Replace log.Fatalf
    def replacer_fatal(match):
        fmt_str = match.group(1)
        args = match.group(2)
        clean_fmt = re.sub(r'\[.*?\]\s*', '', fmt_str)
        clean_fmt = re.sub(r':\s*%[vs]', '', clean_fmt)
        clean_fmt = re.sub(r'\s*%[vs]', '', clean_fmt)
        
        args_list = [a.strip() for a in args.split(',')]
        slog_args = []
        for a in args_list:
            if a == 'err':
                slog_args.append('"error", err')
            else:
                slog_args.append(f'"val", {a}')
                
        return f'slog.Error("{clean_fmt}", {", ".join(slog_args)})\n\t\tos.Exit(1)'
        
    content = re.sub(r'log\.Fatalf\("(.*?)"\s*,\s*(.*?)\)', replacer_fatal, content)
    
    if original != content:
        # Add "log/slog" if not exists
        if '"log/slog"' not in content:
            content = content.replace('import (', 'import (\n\t"log/slog"')
        with open(filepath, 'w') as f:
            f.write(content)
        print(f"Updated {filepath}")

for root, dirs, files in os.walk('internal'):
    for file in files:
        if file.endswith('.go'):
            process_file(os.path.join(root, file))
