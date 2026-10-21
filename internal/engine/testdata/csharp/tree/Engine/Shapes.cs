namespace Loomyard.Shapes;

public class Box<T>
{
    public Box(T value) { Value = value; }

    public T Value { get; }
}

public struct Vector
{
    public int X;
    public int Y;
}

public interface IShape
{
    double Area();
}

public enum Color
{
    Red,
    Green,
}

public delegate void Notify(string message);

public record Point(int X)
{
    public int X { get; init; } = X;
}

public record Pair(int Left, int Right);

public record struct Cell(int Row, int Column);

public class Holder(string name)
{
    public string Name => name;
}
