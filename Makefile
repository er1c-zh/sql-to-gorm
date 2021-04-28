antlr4=java -Xmx500M -cp "/usr/local/lib/antlr-4.9-complete.jar:$CLASSPATH" org.antlr.v4.Tool

gen: clean_gen

	cp -r grammars-v4/sql/mysql/Positive-Technologies/* ./antlr4_gen/

	$(antlr4) ./antlr4_gen/MySqlLexer.g4 -Dlanguage=Go -package antlr4_gen
	$(antlr4) ./antlr4_gen/MySqlParser.g4 -Dlanguage=Go -package antlr4_gen

clean_gen:
	rm -rf ./antlr4_gen/* 
